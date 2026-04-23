# Cliplucid

Cliplucid is a macOS hotkey tool that enables asking (with voice input) your favorite AI about your clipboard.

## Features

Currently, there are three modes:

* Shift+Command+1: Opens a new tab on your default browser, which explains the text in you clipboard
  * Example: You want a fast answer of your AI for an term you don't know. Mark the term, copy it via Command+C or Right click->copy and then press the Shift+Command+1 and you get shortly an answer which gets automaticly opened for you.
* Shift+Ctrl+Command+1: Opens a new tab on your default browser, and shows the answer to your question which you speaked in via voice command. You are able to ask stuff about the things in your clipboard, too. The voice command keeps recorded, while you hold down the hotkeys.
  * Example: You are working in the terminal and you get an error you don't know. Copy the error, then press and hold Shift+Ctrl+Command+1. Now you speak out your question: "What does this error mean? I just wanted to create a file.". When you release the hotkeys, then you'll get shortly an answer, which gets automaticly presented to you.
* Shift+Ctrl+Command+2: Replaces your clipboard with the answer to your question, which you speaked in via voice command.  The voice command keeps recorded, while you hold down the hotkeys.
  * Example: You receive an email and want to let the AI create an answer for you. Copy the text of the received email. Press and hold Shift+Ctrl+Command+2 and give instructions with your voice, like: "Create an answer to this email.". When you release the hotkeys, then after a short time you hear a sound, which let's you know, that the answer for the email is in your clipboard. Click now in the email on the answer button and paste your generated text to it.


## Requirements

- macOS (current version)
- Go 1.22+ (if you want to build this project by yourself)
- Accessibility permission for global hotkeys:
  - `System Settings -> Privacy & Security -> Accessibility`
- Microphone permission for audio capture:
  - `System Settings -> Privacy & Security -> Microphone`
- `sox` for recording (`brew install sox`)
- `whisper-cli` (from `whisper.cpp`) for local speech-to-text
- Available whisper model. See Installation section.

## Installation

First install the dependencies (requires [Homebrew](https://brew.sh/) installed):
```
brew install sox
```
And after that install whisper-cli which comes with [whisper.cpp](https://github.com/ggml-org/whisper.cpp). Please follow their installation guide.

For voice input you need a transcription model. You can download one as follows:
```
curl -L -o ggml-small.bin https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-small.bin
```

Now, you can install cliplucid.
Either download the latest release and run it, or build it via go.
A quick start would be, if you have go installed:
```
go run main.go
```

## Configuration

1. Download the current example.conf:

```bash
curl -L -o ~/.config/cliplucid/config.yaml https://raw.githubusercontent.com/berti92/cliplucid/refs/heads/main/config.example.yaml
```

2. Download the default prompts:
```bash
mkdir -p ~/.config/cliplucid/prompts/
curl -L -o ~/.config/cliplucid/prompts/ask-with-voice.md https://raw.githubusercontent.com/berti92/cliplucid/refs/heads/main/prompts/ask-with-voice.md
curl -L -o ~/.config/cliplucid/prompts/help-from-clipboard.md https://raw.githubusercontent.com/berti92/cliplucid/refs/heads/main/prompts/help-from-clipboard.md
curl -L -o ~/.config/cliplucid/prompts/rewrite-with-voice.md https://raw.githubusercontent.com/berti92/cliplucid/refs/heads/main/prompts/rewrite-with-voice.md
```

2. Edit `~/.config/cliplucid/config.yaml`:

- `api.base_url` (for example `http://localhost:1234/v1`, must be OpenAI compatible)
- `api.model` (for example `google/gemma-4-e4b:2`)
- `voice.language` (for example `de`, `en`, `fr`)
- `voice.whisper_model` (path to a local `ggml-*.bin` model file)
- `actions` (`hotkey`, `mode`, `prompt_file`, optional `done_sound`)

## Optional: You can edit the prompts for the functions
If you are unhappy with the results, then you can adjust the prompts to your needs.
* ~/.config/cliplucid/prompts/ask-with-voice.md
* ~/.config/cliplucid/prompts/help-from-clipboard.md
* ~/.config/cliplucid/prompts/rewrite-with-voice.md

### Prompt Placeholders

- `%CLIPBOARD%`: current clipboard content
- `%VOICECONTEXT%`: speech-to-text result

## Run

If you want to build and run it:
```bash
go run main.go
```

If you want to run the binary:
```bash
./cliplucid
```

The process runs continuously and listens for configured hotkeys.

## License

GPLv3