package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/atotto/clipboard"
	"github.com/yuin/goldmark"
	"golang.design/x/hotkey"
	"golang.design/x/hotkey/mainthread"
	"gopkg.in/yaml.v3"
)

const (
	modePopup          = "popup"
	modePopupVoice     = "popup_voice"
	modeClipboardVoice = "clipboard_voice"
)

type Config struct {
	API struct {
		BaseURL string `yaml:"base_url"`
		APIKey  string `yaml:"api_key"`
		Model   string `yaml:"model"`
	} `yaml:"api"`
	Voice struct {
		Language     string `yaml:"language"`
		WhisperModel string `yaml:"whisper_model"`
	} `yaml:"voice"`
	Actions []ActionConfig `yaml:"actions"`
}

type ActionConfig struct {
	Name       string `yaml:"name"`
	Hotkey     string `yaml:"hotkey"`
	Mode       string `yaml:"mode"`
	PromptFile string `yaml:"prompt_file"`
	DoneSound  string `yaml:"done_sound"`
}

type App struct {
	cfg       Config
	client    *http.Client
	bindings  []*binding
	configDir string
}

type binding struct {
	action ActionConfig
	hk     *hotkey.Hotkey

	mu        sync.Mutex
	recorder  *recordSession
	running   bool
	recording bool
}

type recordSession struct {
	cmd       *exec.Cmd
	audioFile string
	tmpDir    string
}

type chatCompletionsRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionsResponse struct {
	Choices []struct {
		Message struct {
			Content any `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func main() {
	configPath := flag.String("config", "./config.yaml", "Pfad zur Konfigurationsdatei")
	flag.Parse()

	app, err := newApp(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	mainthread.Init(func() {
		if err := app.registerHotkeys(); err != nil {
			log.Fatalf("Failed to register hotkeys: %v", err)
		}

		log.Printf("cliplucid is running with %d hotkeys", len(app.bindings))

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh

		app.unregisterHotkeys()
		log.Println("Stopped")
	})
}

func newApp(configPath string) (*App, error) {
	configAbs, err := resolveConfigPath(configPath)
	if err != nil {
		return nil, err
	}

	raw, err := os.ReadFile(configAbs)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return nil, err
	}

	if cfg.API.BaseURL == "" || cfg.API.Model == "" {
		return nil, errors.New("api.base_url und api.model sind Pflichtfelder")
	}

	if len(cfg.Actions) == 0 {
		return nil, errors.New("mindestens eine Action muss konfiguriert sein")
	}

	if strings.TrimSpace(cfg.Voice.Language) == "" {
		cfg.Voice.Language = "de"
	}

	configDir := filepath.Dir(configAbs)
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("konnte Home-Verzeichnis nicht ermitteln: %w", err)
	}
	promptFallbackRoot := filepath.Join(homeDir, ".config", "cliplucid")

	if hasVoiceActions(cfg.Actions) {
		if strings.TrimSpace(cfg.Voice.WhisperModel) == "" {
			return nil, errors.New("voice.whisper_model ist erforderlich, wenn Voice-Aktionen konfiguriert sind")
		}
		resolvedModelPath, err := resolveFilePath(cfg.Voice.WhisperModel, configDir, promptFallbackRoot)
		if err != nil {
			return nil, fmt.Errorf("voice.whisper_model: %w", err)
		}
		cfg.Voice.WhisperModel = resolvedModelPath
	}

	for i := range cfg.Actions {
		if cfg.Actions[i].Name == "" {
			cfg.Actions[i].Name = fmt.Sprintf("action_%d", i+1)
		}
		if cfg.Actions[i].PromptFile == "" {
			return nil, fmt.Errorf("action %q braucht prompt_file", cfg.Actions[i].Name)
		}
		resolvedPromptPath, err := resolvePromptPath(cfg.Actions[i].PromptFile, configDir, promptFallbackRoot)
		if err != nil {
			return nil, fmt.Errorf("action %q: %w", cfg.Actions[i].Name, err)
		}
		cfg.Actions[i].PromptFile = resolvedPromptPath
		if cfg.Actions[i].DoneSound != "" && !filepath.IsAbs(cfg.Actions[i].DoneSound) {
			cfg.Actions[i].DoneSound = filepath.Join(configDir, cfg.Actions[i].DoneSound)
		}
		switch cfg.Actions[i].Mode {
		case modePopup, modePopupVoice, modeClipboardVoice:
		default:
			return nil, fmt.Errorf("action %q hat ungueltigen mode %q", cfg.Actions[i].Name, cfg.Actions[i].Mode)
		}
	}

	return &App{
		cfg:       cfg,
		configDir: configDir,
		client: &http.Client{
			Timeout: 2 * time.Minute,
		},
	}, nil
}

func resolveConfigPath(configPath string) (string, error) {
	absPath, err := filepath.Abs(configPath)
	if err != nil {
		return "", err
	}

	if _, err := os.Stat(absPath); err == nil {
		return absPath, nil
	} else if !os.IsNotExist(err) {
		return "", err
	}

	isDefaultConfigName := configPath == "config.yaml" || configPath == "./config.yaml"
	if !isDefaultConfigName {
		return "", fmt.Errorf("konfigurationsdatei nicht gefunden: %s", absPath)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("konnte Home-Verzeichnis nicht ermitteln: %w", err)
	}

	fallback := filepath.Join(homeDir, ".config", "cliplucid", "config.yaml")
	if _, err := os.Stat(fallback); err == nil {
		log.Printf("Using fallback configuration: %s", fallback)
		return fallback, nil
	} else if !os.IsNotExist(err) {
		return "", err
	}

	return "", fmt.Errorf("keine Konfiguration gefunden unter %s oder %s", absPath, fallback)
}

func resolvePromptPath(promptFile string, configDir string, fallbackRoot string) (string, error) {
	if filepath.IsAbs(promptFile) {
		if _, err := os.Stat(promptFile); err == nil {
			return promptFile, nil
		} else if os.IsNotExist(err) {
			return "", fmt.Errorf("prompt_file nicht gefunden: %s", promptFile)
		} else {
			return "", err
		}
	}

	primary := filepath.Join(configDir, promptFile)
	if _, err := os.Stat(primary); err == nil {
		return primary, nil
	} else if !os.IsNotExist(err) {
		return "", err
	}

	fallback := filepath.Join(fallbackRoot, promptFile)
	if _, err := os.Stat(fallback); err == nil {
		log.Printf("Using fallback prompt: %s", fallback)
		return fallback, nil
	} else if !os.IsNotExist(err) {
		return "", err
	}

	return "", fmt.Errorf("prompt_file %q nicht gefunden unter %s oder %s", promptFile, primary, fallback)
}

func resolveFilePath(pathValue string, configDir string, fallbackRoot string) (string, error) {
	if filepath.IsAbs(pathValue) {
		if _, err := os.Stat(pathValue); err == nil {
			return pathValue, nil
		} else if os.IsNotExist(err) {
			return "", fmt.Errorf("nicht gefunden: %s", pathValue)
		} else {
			return "", err
		}
	}

	primary := filepath.Join(configDir, pathValue)
	if _, err := os.Stat(primary); err == nil {
		return primary, nil
	} else if !os.IsNotExist(err) {
		return "", err
	}

	fallback := filepath.Join(fallbackRoot, pathValue)
	if _, err := os.Stat(fallback); err == nil {
		log.Printf("Using fallback file: %s", fallback)
		return fallback, nil
	} else if !os.IsNotExist(err) {
		return "", err
	}

	return "", fmt.Errorf("nicht gefunden unter %s oder %s", primary, fallback)
}

func hasVoiceActions(actions []ActionConfig) bool {
	for _, action := range actions {
		if action.Mode == modePopupVoice || action.Mode == modeClipboardVoice {
			return true
		}
	}
	return false
}

func (a *App) registerHotkeys() error {
	for _, action := range a.cfg.Actions {
		mods, key, err := parseHotkey(action.Hotkey)
		if err != nil {
			return fmt.Errorf("action %q: %w", action.Name, err)
		}

		hk := hotkey.New(mods, key)
		if err := hk.Register(); err != nil {
			return fmt.Errorf("action %q (%s): %w", action.Name, action.Hotkey, err)
		}

		b := &binding{action: action, hk: hk}
		a.bindings = append(a.bindings, b)

		go a.runBinding(b)
		log.Printf("Registered hotkey: %-20s -> %s", action.Name, action.Hotkey)
	}

	return nil
}

func (a *App) unregisterHotkeys() {
	for _, b := range a.bindings {
		_ = b.hk.Unregister()
	}
}

func (a *App) runBinding(b *binding) {
	for {
		select {
		case <-b.hk.Keydown():
			a.onKeyDown(b)
		case <-b.hk.Keyup():
			a.onKeyUp(b)
		}
	}
}

func (a *App) onKeyDown(b *binding) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.running {
		return
	}

	switch b.action.Mode {
	case modePopup:
		b.running = true
		go a.runActionNoVoice(b)
	case modePopupVoice, modeClipboardVoice:
		if b.recording {
			return
		}
		rec, err := a.startRecording()
		if err != nil {
			log.Printf("%s: Failed to start recording: %v", b.action.Name, err)
			a.showErrorWindow("Aufnahmefehler", err.Error())
			return
		}
		b.recorder = rec
		b.recording = true
		log.Printf("%s: Recording started", b.action.Name)
	}
}

func (a *App) onKeyUp(b *binding) {
	b.mu.Lock()
	if b.action.Mode == modePopup {
		b.mu.Unlock()
		return
	}
	if !b.recording || b.running || b.recorder == nil {
		b.mu.Unlock()
		return
	}
	b.running = true
	rec := b.recorder
	b.recorder = nil
	b.recording = false
	b.mu.Unlock()

	go a.runActionWithVoice(b, rec)
}

func (a *App) runActionNoVoice(b *binding) {
	defer func() {
		b.mu.Lock()
		b.running = false
		b.mu.Unlock()
	}()

	if err := a.executeAction(b.action, ""); err != nil {
		log.Printf("%s: %v", b.action.Name, err)
		a.showErrorWindow("Fehler", err.Error())
	}
}

func (a *App) runActionWithVoice(b *binding, rec *recordSession) {
	defer func() {
		b.mu.Lock()
		b.running = false
		b.mu.Unlock()
	}()

	text, err := a.stopAndTranscribe(rec)
	if err != nil {
		log.Printf("%s: %v", b.action.Name, err)
		a.showErrorWindow("Transkriptionsfehler", err.Error())
		return
	}

	if err := a.executeAction(b.action, text); err != nil {
		log.Printf("%s: %v", b.action.Name, err)
		a.showErrorWindow("Fehler", err.Error())
	}
}

func (a *App) executeAction(action ActionConfig, voiceContext string) error {
	clipboardContent, err := clipboard.ReadAll()
	if err != nil {
		return fmt.Errorf("clipboard konnte nicht gelesen werden: %w", err)
	}

	promptRaw, err := os.ReadFile(action.PromptFile)
	if err != nil {
		return fmt.Errorf("prompt_file %q konnte nicht gelesen werden: %w", action.PromptFile, err)
	}

	prompt := strings.ReplaceAll(string(promptRaw), "%CLIPBOARD%", clipboardContent)
	prompt = strings.ReplaceAll(prompt, "%VOICECONTEXT%", voiceContext)

	answer, err := a.callChatCompletions(prompt)
	if err != nil {
		return err
	}

	switch action.Mode {
	case modePopup, modePopupVoice:
		a.showMarkdownWindow(action.Name, answer)
	case modeClipboardVoice:
		if err := clipboard.WriteAll(answer); err != nil {
			return fmt.Errorf("antwort konnte nicht ins Clipboard geschrieben werden: %w", err)
		}
		if action.DoneSound != "" {
			if err := playSound(action.DoneSound); err != nil {
				log.Printf("%s: Failed to play completion sound: %v", action.Name, err)
			}
		}
	}

	return nil
}

func (a *App) startRecording() (*recordSession, error) {
	if _, err := exec.LookPath("sox"); err != nil {
		return nil, errors.New("sox wurde nicht gefunden. Bitte installiere sox (brew install sox)")
	}

	tmpDir, err := os.MkdirTemp("", "cliplucid-audio-*")
	if err != nil {
		return nil, err
	}

	rec := &recordSession{
		audioFile: filepath.Join(tmpDir, "recording.wav"),
		tmpDir:    tmpDir,
	}

	cmd := exec.Command("sox", "-q", "-d", rec.audioFile, "rate", "16000", "channels", "1")
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		_ = os.RemoveAll(tmpDir)
		return nil, err
	}

	rec.cmd = cmd
	return rec, nil
}

func (a *App) stopAndTranscribe(rec *recordSession) (string, error) {
	defer os.RemoveAll(rec.tmpDir)

	if rec.cmd.Process != nil {
		_ = rec.cmd.Process.Signal(os.Interrupt)
	}

	done := make(chan error, 1)
	go func() {
		done <- rec.cmd.Wait()
	}()

	select {
	case <-time.After(5 * time.Second):
		if rec.cmd.Process != nil {
			_ = rec.cmd.Process.Kill()
		}
		<-done
	case err := <-done:
		if err != nil {
			return "", fmt.Errorf("record_command ist fehlgeschlagen: %w", err)
		}
	}

	if _, err := exec.LookPath("whisper-cli"); err != nil {
		return "", errors.New("whisper-cli wurde nicht gefunden. Bitte installiere whisper-cpp")
	}

	outputBase := filepath.Join(rec.tmpDir, "transcript")
	txCmd := exec.Command(
		"whisper-cli",
		"-m", a.cfg.Voice.WhisperModel,
		"-f", rec.audioFile,
		"-l", a.cfg.Voice.Language,
		"-otxt",
		"-of", outputBase,
	)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	txCmd.Stdout = &stdout
	txCmd.Stderr = &stderr

	if err := txCmd.Run(); err != nil {
		return "", fmt.Errorf("transcribe_command ist fehlgeschlagen: %w (%s)", err, strings.TrimSpace(stderr.String()))
	}

	raw, err := os.ReadFile(outputBase + ".txt")
	if err != nil {
		return "", fmt.Errorf("konnte Whisper-Ausgabe nicht lesen: %w", err)
	}

	text := strings.TrimSpace(string(raw))
	if text == "" {
		return "", fmt.Errorf("Transkription hat keinen Text geliefert (%s.txt leer)", outputBase)
	} else {
		log.Printf("Transcription: %v", text)
	}

	return text, nil
}

func (a *App) callChatCompletions(prompt string) (string, error) {
	url := strings.TrimRight(a.cfg.API.BaseURL, "/") + "/chat/completions"
	reqBody := chatCompletionsRequest{
		Model: a.cfg.API.Model,
		Messages: []chatMessage{
			{Role: "user", Content: prompt},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if a.cfg.API.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+a.cfg.API.APIKey)
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("api-call fehlgeschlagen: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("api-call fehlgeschlagen (%d): %s", resp.StatusCode, string(raw))
	}

	var data chatCompletionsResponse
	if err := json.Unmarshal(raw, &data); err != nil {
		return "", fmt.Errorf("ungueltige API-Antwort: %w", err)
	}
	if len(data.Choices) == 0 {
		return "", errors.New("API hat keine Choices geliefert")
	}

	content := parseContent(data.Choices[0].Message.Content)
	if strings.TrimSpace(content) == "" {
		return "", errors.New("API hat leeren Content geliefert")
	}

	return content, nil
}

func parseContent(v any) string {
	switch c := v.(type) {
	case string:
		return c
	case []any:
		var b strings.Builder
		for _, part := range c {
			obj, ok := part.(map[string]any)
			if !ok {
				continue
			}
			if t, _ := obj["type"].(string); t != "text" {
				continue
			}
			txt, _ := obj["text"].(string)
			b.WriteString(txt)
		}
		return b.String()
	default:
		return ""
	}
}

func (a *App) showMarkdownWindow(title string, markdown string) {
	go func() {
		var rendered bytes.Buffer
		if err := goldmark.Convert([]byte(markdown), &rendered); err != nil {
			rendered.WriteString("<pre>Markdown rendering failed.</pre>")
		}

		html := "<!doctype html><html><head><meta charset=\"utf-8\"><meta name=\"viewport\" content=\"width=device-width,initial-scale=1\"><title>" + escapeHTML(title) + "</title><style>body{font-family:-apple-system,BlinkMacSystemFont,\"Segoe UI\",sans-serif;margin:0;background:#f4f2ec;color:#1f1a14;}main{max-width:880px;margin:0 auto;padding:28px 24px 48px;line-height:1.6;}pre{background:#f0ead8;padding:12px;border-radius:10px;overflow:auto;}code{background:#ece4ce;padding:2px 6px;border-radius:6px;}blockquote{border-left:4px solid #9a7f50;padding-left:14px;color:#5e4d33;}h1,h2,h3{line-height:1.2;}table{border-collapse:collapse;}th,td{border:1px solid #d7cfbb;padding:8px;}</style></head><body><main>" + rendered.String() + "</main></body></html>"

		dir, err := os.MkdirTemp("", "cliplucid-popup-*")
		if err != nil {
			log.Printf("Failed to create popup: %v", err)
			return
		}
		path := filepath.Join(dir, "response.html")
		if err := os.WriteFile(path, []byte(html), 0o644); err != nil {
			log.Printf("Failed to write popup: %v", err)
			return
		}

		openCmd := exec.Command("open", path)
		if err := openCmd.Start(); err != nil {
			log.Printf("Failed to open popup: %v", err)
		}
	}()
}

func (a *App) showErrorWindow(title string, message string) {
	a.showMarkdownWindow(title, "## "+title+"\n\n"+message)
}

func playSound(path string) error {
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("done_sound nicht gefunden: %w", err)
	}
	cmd := exec.Command("afplay", path)
	return cmd.Start()
}

func escapeHTML(s string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		"\"", "&quot;",
		"'", "&#39;",
	)
	return replacer.Replace(s)
}

func parseHotkey(input string) ([]hotkey.Modifier, hotkey.Key, error) {
	parts := strings.Split(strings.ToLower(strings.TrimSpace(input)), "+")
	if len(parts) < 2 {
		return nil, 0, fmt.Errorf("ungueltiges Hotkey-Format %q, Beispiel: cmd+shift+e", input)
	}

	var mods []hotkey.Modifier
	for _, p := range parts[:len(parts)-1] {
		switch strings.TrimSpace(p) {
		case "cmd", "command", "super":
			mods = append(mods, hotkey.ModCmd)
		case "ctrl", "control", "ctl":
			mods = append(mods, hotkey.ModCtrl)
		case "alt", "option", "opt":
			mods = append(mods, hotkey.ModOption)
		case "shift":
			mods = append(mods, hotkey.ModShift)
		default:
			return nil, 0, fmt.Errorf("unbekannter Modifier %q", p)
		}
	}

	keyName := strings.TrimSpace(parts[len(parts)-1])
	key, ok := hotkeyKeyMap()[keyName]
	if !ok {
		return nil, 0, fmt.Errorf("unbekannte Taste %q", keyName)
	}

	return mods, key, nil
}

func hotkeyKeyMap() map[string]hotkey.Key {
	return map[string]hotkey.Key{
		"a": hotkey.KeyA, "b": hotkey.KeyB, "c": hotkey.KeyC, "d": hotkey.KeyD,
		"e": hotkey.KeyE, "f": hotkey.KeyF, "g": hotkey.KeyG, "h": hotkey.KeyH,
		"i": hotkey.KeyI, "j": hotkey.KeyJ, "k": hotkey.KeyK, "l": hotkey.KeyL,
		"m": hotkey.KeyM, "n": hotkey.KeyN, "o": hotkey.KeyO, "p": hotkey.KeyP,
		"q": hotkey.KeyQ, "r": hotkey.KeyR, "s": hotkey.KeyS, "t": hotkey.KeyT,
		"u": hotkey.KeyU, "v": hotkey.KeyV, "w": hotkey.KeyW, "x": hotkey.KeyX,
		"y": hotkey.KeyY, "z": hotkey.KeyZ,
		"0": hotkey.Key0, "1": hotkey.Key1, "2": hotkey.Key2, "3": hotkey.Key3,
		"4": hotkey.Key4, "5": hotkey.Key5, "6": hotkey.Key6, "7": hotkey.Key7,
		"8": hotkey.Key8, "9": hotkey.Key9,
		"f1": hotkey.KeyF1, "f2": hotkey.KeyF2, "f3": hotkey.KeyF3, "f4": hotkey.KeyF4,
		"f5": hotkey.KeyF5, "f6": hotkey.KeyF6, "f7": hotkey.KeyF7, "f8": hotkey.KeyF8,
		"f9": hotkey.KeyF9, "f10": hotkey.KeyF10, "f11": hotkey.KeyF11, "f12": hotkey.KeyF12,
		"space":  hotkey.KeySpace,
		"enter":  hotkey.KeyReturn,
		"return": hotkey.KeyReturn,
		"tab":    hotkey.KeyTab,
		"esc":    hotkey.KeyEscape,
		"escape": hotkey.KeyEscape,
		"up":     hotkey.KeyUp,
		"down":   hotkey.KeyDown,
		"left":   hotkey.KeyLeft,
		"right":  hotkey.KeyRight,
	}
}
