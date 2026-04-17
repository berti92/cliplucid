# Spezifikation
## Background
KI bzw. LLMs erleichtern den Alltag und werden oft benutzt. Es ist leider so, dass diese meistens in irgendwelche Chat-Webseiten integriert sind. 
Dadurch hat man immer einen Bruch im Workflow. 

Ich stelle kurz mein Problem dar: 

Ich arbeite an meinem Computer (Mac OS, aktuelle Version). Ich bin Software-Entwickler.
Ich habe nun das Terminal offen und führe einen Befehl aus. Der Befehl liefert eine mir unbekannte Fehlermeldung.
Wenn ich nun die Lösung für dieses Problem recherchieren möchte, muss ich den Output des Befehls kopieren.
Die Webseite eines beliebigen LLM-Providers aufrufen und dort den Output reinkopieren und beschreiben, was ich
gemacht habe und was mein Ziel ist.
D.h. ich musste hier nun kopieren, Browser öffnen, LLM-Website aufrufen, einfügen und weiter mein Problem beschreiben.

## Ziel

Es soll ein Programm geschrieben, dass permanent läuft.
Das Programm lauscht auf eine verschiedene Tastenkombinationen.
Die genaue Tastenkombinationen sollen in einer Konfiguration editierbar sein.
Jede Kombination kann eine der nachfolgenden Funktionen aufrufen.

### 1. Funktion
Macht einen Api-Call zu einer lokalen API (OpenAI-comptabile v1 chat completions).
Führt einen Prompt aus der konfigurierbar ist, das ist ein eigenes File auf dem Filesystem,
dass den kompletten Prompt enthält in Markdown. In dem Markdown gibt es eine Textmarke `%CLIPBOARD%`.
Diese Textmarke wird ersetzt mit dem Inhalt aus der Zwischenablage bzw. Clipboard auf meinem Computer.
Der o.g. API-Call wird mit dem o.g. Prompt abgesetzt und wenn eine Antwort zurückgekommen ist, wird die
Antwort davon in einem Hinweisfenster angezeigt. Das Hinweisfenster muss Markdown darstellen können, denn
so wird die Antwort gestaltet sein.

### 2. Funktion
Macht einen Api-Call zu einer lokalen API (OpenAI-comptabile v1 chat completions).
Führt einen Prompt aus der konfigurierbar ist, das ist ein eigenes File auf dem Filesystem,
dass den kompletten Prompt enthält in Markdown. In dem Markdown gibt es eine Textmarke `%CLIPBOARD%`.
Diese Textmarke wird ersetzt mit dem Inhalt aus der Zwischenablage bzw. Clipboard auf meinem Computer.
Es gibt eine weitere Textmarke `%VOICECONTEXT%`. In diese Textmarke soll der Voice to Text von meinem Mikrofon
aufgenommen werden und hier als Text eingesetzt werden. Die Länge der Aufnahme richtet sich danach, wie lange
ich die Tastenkombination halte. Wenn ich diese loslasse, wird meine Aufnahme beendet und das Speech to text ausgeführt und 
in die Textmarke eingesetzt. Finde selber heraus, welche TTS-Möglichkeiten es gibt, einzige Einschränkung ist, dass
das auf meinem PC (aktuelles Mac OS) ausgeführt werden kann.
Der o.g. API-Call wird mit dem o.g. Prompt abgesetzt und wenn eine Antwort zurückgekommen ist, wird die
Antwort davon in einem Hinweisfenster angezeigt. Das Hinweisfenster muss Markdown darstellen können, denn
so wird die Antwort gestaltet sein.

### 3. Funktion
Macht einen Api-Call zu einer lokalen API (OpenAI-comptabile v1 chat completions).
Führt einen Prompt aus der konfigurierbar ist, das ist ein eigenes File auf dem Filesystem,
dass den kompletten Prompt enthält in Markdown. In dem Markdown gibt es eine Textmarke `%CLIPBOARD%`.
Diese Textmarke wird ersetzt mit dem Inhalt aus der Zwischenablage bzw. Clipboard auf meinem Computer.
Es gibt eine weitere Textmarke `%VOICECONTEXT%`. In diese Textmarke soll der Voice to Text von meinem Mikrofon
aufgenommen werden und hier als Text eingesetzt werden. Die Länge der Aufnahme richtet sich danach, wie lange
ich die Tastenkombination halte. Wenn ich diese loslasse, wird meine Aufnahme beendet und das Speech to text ausgeführt und 
in die Textmarke eingesetzt. Finde selber heraus, welche TTS-Möglichkeiten es gibt, einzige Einschränkung ist, dass
das auf meinem PC (aktuelles Mac OS) ausgeführt werden kann.
Der o.g. API-Call wird mit dem o.g. Prompt abgesetzt und wenn eine Antwort zurückgekommen ist, wird die
Antwort davon in die Zwischenablage / Clipboard meines PCs abgelegt. Wenn das fertig ist, soll
ein Hinweiston ertönen, der konfigurierbar ist in der Konfigurationsdatei. Da soll der Pfad zu dem Hinweiston
gekennzeichnet sein.

## Regeln
* Nehme go für deine Entwicklung.