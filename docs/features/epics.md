# **EP-001 — Repository & Tech Baseline (Go + TUI Skeleton)**

**Goal:** Monorepo-Skelett und verbindliche Technikentscheidungen (Go, Bubble Tea) für eine statisch gebaute, headless CLI/TUI schaffen.

**Capabilities:**

* Single-Binary Build (Go), Cross-Compile, statisch gelinkt.
* TUI-Grundgerüst (Bubble Tea, Lip Gloss) mit Tastaturbedienung.
* Modulstruktur für Installer, Services, Power, Metrics, Diag, Update.

**Services/Endpoints:** *n/a*

**Data Contracts:**

* `repo-structure.md` (Ordnerstruktur, Build-Ziele).
* `styleguide.md` (Logging-Levels, Error-Handling-Prinzipien).

**Acceptance (DoD):**

* Given ein frisches Clone, When `make build` läuft, Then entsteht eine statische `aistack`-Binary.
* Given ein SSH-Terminal, When `./aistack` startet, Then zeigt die TUI ein leeres Hauptmenü ohne Panic.
* Given die Modulstruktur, When `go test ./...` läuft, Then alle Basistests grün ≥ 80% für Core-Pakete.

**Risks/Dependencies:**

* Abhängigkeiten von Bubble Tea-Versionen.
* Cross-Compile Mac/Win bewusst out-of-scope (v1 Linux-only).

**Solution Proposal:**

* Go 1.x LTS; Module unter `cmd/aistack`, `internal/*`.
* Bubble Tea + Lip Gloss; ANSI-Color nur, keine Maus.
* Makefile mit Targets: build, test, lint, release.
* Lint via `golangci-lint`; Errors als `stderr` JSON-Logs.

## Stories

### Story T-001 — Repository-Skelett & Build-Pipeline (lokal)

**User Story:** Als Entwickler:in möchte ich ein minimales Repo mit Go-Modul und Makefile, damit ich lokal reproduzierbar eine statische Binary bauen kann.

**Scope**

* **In scope:** Go-Modul-Init, Makefile mit Targets (`build`, `test`, `lint`), statische Binary.
* **Out of scope:** CI/CD, TUI-Screens.

**Dependencies & Order**

* None.

**Contracts & Data**

* **Data Model:** `repo-structure.md` (Dokumentation der Ordner).
* **Storage:** Verzeichnisse `cmd/aistack`, `internal/*`, `assets/`.

**States & Error Cases**

* Build fehlend: Go nicht installiert.
* Lint-Fehler: Abbruch mit Exit-Code ≠ 0.
* Fehlende Abhängigkeiten: Mod-Download schlägt fehl (Netzwerk).

**Solution Proposal (technical guide)**

* Go-Modul initialisieren, `-tags netgo`/`CGO_ENABLED=0` für statische Binary.
* Makefile-Targets mit klaren Phony-Targets.
* `golangci-lint` als lokaler Check.

**Acceptance Criteria (Gherkin-like)**

1. Given ein frisches Clone, When `make build` läuft, Then entsteht `./dist/aistack` ausführbar.
2. Given das Repo, When `make test` läuft, Then alle Tests grün mit Exit-Code 0.
3. Given das Repo, When `make lint` läuft, Then keine Lint-Fehler.

**Test Plan**

* **Unit:** Keine (nur Build).
* **Integration:** Smoke-Build.
* **Fixtures:** n/a.
* **Coverage:** n/a.

**Migration / Compatibility**

* n/a.

**Feature Flag / Config (if relevant)**

* n/a.

**Risks & Mitigations**

* Fehlende Tooling-Versionen → README-Abschnitt „Prereqs“.

**Open Questions**

* None.

**Definition of Done (DoD)**

* Build/Lint/Test laufen lokal grün, Binary statisch.

### Story T-002 — TUI-Bootstrap ohne Screens (Bubble Tea)

**User Story:** Als Nutzer:in möchte ich `aistack` starten und einen leeren TUI-Rahmen sehen, damit die App interaktiv starten kann.

**Scope**

* **In scope:** Main-Loop, Quit via `q`/`Ctrl+C`, Farbschema.
* **Out of scope:** Menüs/Navigation.

**Dependencies & Order**

* Depends on T-001.

**Contracts & Data**

* **Events:**

  | Type          | Payload         | Trigger | Guarantees         | Observability |
    | ------------- | --------------- | ------- | ------------------ | ------------- |
  | `app.started` | `{ts, version}` | Start   | Einmalig pro Start | Log `info`    |
  | `app.exited`  | `{ts, reason}`  | Exit    | Einmalig pro Exit  | Log `info`    |

**States & Error Cases**

* Terminal ohne ANSI → fallback monochrom.
* Resize-Events gehandhabt.

**Solution Proposal (technical guide)**

* Bubble Tea Model mit minimalem `Init/Update/View`.
* Lip Gloss Theme (hoher Kontrast).

**Acceptance Criteria (Gherkin-like)**

1. Given `aistack` gestartet, Then ein leerer Rahmen mit Titel „aistack“ erscheint.
2. Given TUI sichtbar, When Taste `q`, Then Programm beendet sich mit Exit-Code 0.
3. Given Start/Exit, Then `app.started` und `app.exited` werden geloggt.

**Test Plan**

* **Unit:** Update-Loop reagiert auf `q`.
* **Integration:** Starten/Beenden Smoke-Test.

**Migration / Compatibility**

* n/a.

**Feature Flag / Config**

* n/a.

**Risks & Mitigations**

* Nicht-UTF8-Terminals → ASCII-Fallback.

**Open Questions**

* None.

**Definition of Done (DoD)**

* Start/Exit stabil, Logs vorhanden.


# **EP-002 — Bootstrap & System Integration (install.sh + systemd)**

**Goal:** Headless-Bootstrap-Skript richtet Laufzeit, Nutzer/Gruppen, Ordner, systemd-Units und Timer ein.

**Capabilities:**

* `install.sh` prüft Ubuntu 24.04, Internet, sudo.
* Installation Docker (Default) oder Erkennung Podman.
* systemd-Units/-Timer/Udev-Rules deployen/aktivieren.

**Services/Endpoints:** *n/a*

**Data Contracts:**

* `assets/systemd/*.service|*.timer` (Unit-Templates).
* `assets/udev/70-aistack-wol.rules`.

**Acceptance (DoD):**

* Given Ubuntu 24.04, When `curl ... | sudo bash`, Then Docker installiert/aktiviert oder als vorhanden erkannt.
* Given erfolgreicher Bootstrap, Then `systemctl status aistack-agent` ist `active (running)`.
* Given Re-Run des Installers, Then idempotentes Verhalten (keine doppelten Units, keine Fehler).

**Risks/Dependencies:**

* Benutzerumgebung (headless/SSH) ohne interaktive Prompts.
* Docker vs. Podman Parität.

**Solution Proposal:**

* Idempotente Checks (Datei/Unit existiert? Version?).
* `systemctl enable --now` für Agent und Timer.
* Logrotate-Regeln unter `/etc/logrotate.d/aistack`.
* Sudo minimal; Rest unter Gruppe `aistack`.

## Stories

### Story T-003 — Bootstrap-Skript: System-Checks & Docker-Installation

**User Story:** Als Admin möchte ich ein Skript ausführen, das Ubuntu 24.04 prüft und Docker installiert/aktiviert, damit Container laufen.

**Scope**

* **In scope:** OS-Version-Check, sudo-Check, Internet-Check, Docker-Install/Enable.
* **Out of scope:** Podman, systemd-Units.

**Dependencies & Order**

* None.

**Contracts & Data**

* **Events:**

  | Type                         | Payload                | Trigger            | Guarantees          | Observability |
    | ---------------------------- | ---------------------- | ------------------ | ------------------- | ------------- |
  | `bootstrap.checks`           | `{os, sudo, internet}` | Skriptstart        | Vollständige Checks | Log `info`    |
  | `bootstrap.docker.installed` | `{version}`            | Docker installiert | Genau einmal        | Log `info`    |

**States & Error Cases**

* Nicht Ubuntu 24.04 → Abbruch mit klarer Meldung.
* Kein sudo → Abbruch.
* Kein Internet → Abbruch.

**Solution Proposal (technical guide)**

* `lsb_release`/`/etc/os-release` prüfen.
* `systemctl enable --now docker`.

**Acceptance Criteria (Gherkin-like)**

1. Given Ubuntu 24.04, When Skript läuft, Then Docker ist `active (running)`.
2. Given Docker bereits vorhanden, When Skript läuft, Then keine Neuinstallation (idempotent) und Exit 0.
3. Given fehlendes sudo, Then Exit ≠ 0 mit Fehlermeldung.

**Test Plan**

* **Integration:** Container „hello-world“ pull/run.
* **Fixtures:** Mock für OS-Version (dry-run).
* **Coverage:** Erfolg/Fehlerpfade.

**Migration / Compatibility**

* n/a.

**Feature Flag / Config**

* n/a.

**Risks & Mitigations**

* Paketmirror langsam → Timeout & Retry.

**Open Questions**

* None.

**Definition of Done (DoD)**

* Docker zuverlässig installiert/erkannt.

### Story T-004 — Deploy systemd-Agent & Timer (Idle Evaluator Platzhalter)

**User Story:** Als Admin möchte ich, dass das Skript Agent/Timer-Units installiert/aktiviert, damit Hintergrundaufgaben laufen.

**Scope**

* **In scope:** `aistack-agent.service`, `aistack-idle.timer` (nur Platzhalter), Logrotate-Regeln.
* **Out of scope:** Idle-Logik.

**Dependencies & Order**

* Depends on T-003.

**Contracts & Data**

* **Storage:** `/etc/systemd/system/aistack-agent.service`, `/etc/logrotate.d/aistack`.

**States & Error Cases**

* systemd nicht verfügbar → Abbruch mit Hinweis.
* Unit existiert → Replace idempotent.

**Solution Proposal (technical guide)**

* Units aus `assets/systemd` kopieren, `systemctl daemon-reload`, `enable --now`.

**Acceptance Criteria (Gherkin-like)**

1. Given Bootstrap, Then `aistack-agent.service` ist `active (running)`.
2. Given erneute Ausführung, Then keine Duplikate; Status unverändert grün.
3. Given `logrotate -f`, Then Logs werden rotiert ohne Fehler.

**Test Plan**

* **Integration:** `systemctl status`, Log-Rotation Smoke.

**Migration / Compatibility**

* n/a.

**Feature Flag / Config**

* n/a.

**Risks & Mitigations**

* Berechtigungen → Installer läuft via sudo.

**Open Questions**

* None.

**Definition of Done (DoD)**

* Units aktiv, idempotent.


# **EP-003 — Container Runtime & Compose Assets**

**Goal:** Container-Orchestrierung für Ollama, Open WebUI, LocalAI via Docker Compose (Podman optional) bereitstellen.

**Capabilities:**

* Compose-Templates pro Service.
* Gemeinsames Netzwerk `aistack-net`.
* Volume-Management für Daten/Modelle.

**Services/Endpoints:**

* Ollama: `:11434`
* Open WebUI: `:3000` (spricht standardmäßig mit Ollama)
* LocalAI: `:8080`

**Data Contracts:**

* `compose/ollama.yaml`, `compose/openwebui.yaml`, `compose/localai.yaml` (Ports, Volumes, HEALTHCHECK).
* `versions.lock` (optionales Pinnen von Image-Tags).

**Acceptance (DoD):**

* Given `aistack install --profile standard-gpu`, Then alle drei Services laufen und HEALTHCHECKS werden grün.
* Given Podman-System, Then Compose-Äquivalente funktionieren oder sind sauber deaktiviert mit Hinweis (Assumption).

**Risks/Dependencies:**

* GPU-Passthrough (NVIDIA Container Toolkit).
* Portkollisionen 11434/3000/8080.

**Solution Proposal:**

* Default Docker; Podman nur, wenn eindeutig erkannt.
* Restart-Policy `unless-stopped`; HEALTHCHECK HTTP.
* Klare Volumes: `ollama_data`, `openwebui_data`, `localai_models`.

## Stories

### Story T-005 — Compose-Template: Netzwerk & Volumes (aistack-net)

**User Story:** Als Betreiber:in möchte ich ein gemeinsames Netzwerk und dedizierte Volumes, damit Services isoliert und persistent laufen.

**Scope**

* **In scope:** Docker-Netz `aistack-net`, Volumes `ollama_data`, `openwebui_data`, `localai_models`.
* **Out of scope:** Service-Containerdefinitionen.

**Dependencies & Order**

* Depends on T-003.

**Contracts & Data**

* **Data Model:** `compose/common.yaml` mit Netz/Volumes.
* **Guarantees:** Netzname stabil; Volumes nicht gelöscht bei Recreate.

**States & Error Cases**

* Existierendes Netz → Wiederverwenden.
* Konfliktierender Netzname → Fehlermeldung.

**Solution Proposal (technical guide)**

* `docker network create` idempotent; Compose-Teil einbinden.

**Acceptance Criteria (Gherkin-like)**

1. Given kein Netz, When `aistack net up`, Then `aistack-net` existiert.
2. Given Netz existiert, When erneut ausgeführt, Then Exit 0 ohne Änderungen.
3. Given Volumes angelegt, Then `docker volume ls` zeigt alle drei Volumes.

**Test Plan**

* **Integration:** CLI-Aufruf, Docker-Inspektion.
* **Coverage:** Create/Exists.

**Migration / Compatibility**

* n/a.

**Feature Flag / Config**

* n/a.

**Risks & Mitigations**

* Namenskollisionen → Präfix `aistack-`.

**Open Questions**

* None.

**Definition of Done (DoD)**

* Netz/Volumes wiederholbar vorhanden.

### Story T-006 — Compose-Template: Ollama Service (Health & Ports)

**User Story:** Als Nutzer:in möchte ich Ollama per Compose starten, damit die API unter Port 11434 erreichbar ist.

**Scope**

* **In scope:** Service-Definition, HEALTHCHECK, Port 11434, Volume-Bind.
* **Out of scope:** Model-Management.

**Dependencies & Order**

* Depends on T-005.

**Contracts & Data**

* **API Contracts:**

  | Method | Path        | Request Schema | Response Schema  | Status/Error Codes | Guarantees    |
    | ------ | ----------- | -------------- | ---------------- | ------------------ | ------------- |
  | GET    | `/api/tags` | –              | `{models:[...]}` | 200/5xx            | 200 ⇒ healthy |

**States & Error Cases**

* Image-Pull fail → Start schlägt fehl.
* Port belegt → Startfehler.

**Solution Proposal (technical guide)**

* Compose mit `restart: unless-stopped`, HEALTHCHECK GET `/api/tags`.

**Acceptance Criteria (Gherkin-like)**

1. Given `aistack install ollama`, Then Container läuft und `GET /api/tags` liefert 200.
2. Given Port 11434 belegt, Then Fehlermeldung nennt Konflikt-Port.
3. Given Restart des Hosts, Then Ollama startet automatisch.

**Test Plan**

* **Integration:** Portprobe, Health-Endpoint.
* **Fixtures:** Simulierter Portkonflikt (Dummy-Server).

**Migration / Compatibility**

* n/a.

**Feature Flag / Config**

* n/a.

**Risks & Mitigations**

* Netzwerkausfall → Retry Image-Pull.

**Open Questions**

* None.

**Definition of Done (DoD)**

* Ollama läuft stabil, Health grün.

### Story T-007 — Compose-Template: Open WebUI mit Backend-Binding

**User Story:** Als Nutzer:in möchte ich Open WebUI per Compose starten und an ein Backend (Ollama/LocalAI) binden können.

**Scope**

* **In scope:** Service-Definition, HEALTHCHECK, Port 3000, Backend-URL als Config.
* **Out of scope:** Backend-Umschaltung per TUI (separat).

**Dependencies & Order**

* Depends on T-005.

**Contracts & Data**

* **Data Model:** `ui_binding.json` ⇒ `{ active_backend: "ollama|localai", url }`.
* **API Contracts:** HTTP 200 auf `/`.

**States & Error Cases**

* Backend down → UI läuft, aber zeigt Verbindungsfehler (erwartet).

**Solution Proposal (technical guide)**

* Env `OPENWEBUI_BACKEND_URL` setzen; Health: GET `/`.

**Acceptance Criteria (Gherkin-like)**

1. Given Start, Then `GET :3000/` liefert 200.
2. Given Backend-URL geändert, Then Neustart übernimmt URL.
3. Given Backend nicht erreichbar, Then Health gelb (log), UI startet trotzdem.

**Test Plan**

* **Integration:** Start/Stop, Env-Change.
* **Fixtures:** Dummy-Backend.

**Migration / Compatibility**

* n/a.

**Feature Flag / Config**

* n/a.

**Risks & Mitigations**

* Portkollision → klare Meldung.

**Open Questions**

* None.

**Definition of Done (DoD)**

* UI läuft, Backend-URL konfigurierbar.

### Story T-008 — Compose-Template: LocalAI Service (Health & Volume)

**User Story:** Als Nutzer:in möchte ich LocalAI per Compose starten, damit ich alternative Modelle nutzen kann.

**Scope**

* **In scope:** Service-Definition, HEALTHCHECK, Port 8080, Volume `localai_models`.
* **Out of scope:** Modell-Downloads.

**Dependencies & Order**

* Depends on T-005.

**Contracts & Data**

* **API Contracts:**

  | Method | Path       | Request | Response | Status | Guarantees    |
    | ------ | ---------- | ------- | -------- | ------ | ------------- |
  | GET    | `/healthz` | –       | `ok`     | 200    | 200 ⇒ healthy |

**States & Error Cases**

* Image inkompatibel → Startfehler.

**Solution Proposal (technical guide)**

* Health GET `/healthz`, `restart: unless-stopped`.

**Acceptance Criteria (Gherkin-like)**

1. Given Start, Then `GET :8080/healthz` liefert 200.
2. Given Stop, Then Port 8080 frei.
3. Given Host-Reboot, Then Service autostartet.

**Test Plan**

* **Integration:** Health, Restart-Verhalten.

**Migration / Compatibility**

* n/a.

**Feature Flag / Config**

* n/a.

**Risks & Mitigations**

* CPU-only-Fallback vermeiden (GPU in späteren Stories).

**Open Questions**

* None.

**Definition of Done (DoD)**

* LocalAI läuft, Health grün.


# **EP-004 — NVIDIA Stack Detection & Enablement**

**Goal:** NVIDIA-Treiber/CUDA/NVML Präsenz erkennen und passende Container-Fähigkeiten bereitstellen.

**Capabilities:**

* GPU-Erkennung (RTX 4090 bestätigt).
* Prüfung Treiber-/CUDA-Kompatibilität zum Container-Stack.
* Optionaler Install-Hinweis, kein automatisches Kernel-Treiber-Upgrade.

**Services/Endpoints:** *n/a*

**Data Contracts:**

* `gpu_report.json`: `{ driver_version, cuda_compat, nvml_ok, gpus: [...] }`.

**Acceptance (DoD):**

* Given vorhandener Treiber, Then NVML-Calls funktionieren.
* Given inkompatible CUDA, Then TUI zeigt klare Anleitung/Link und blockiert GPU-Workloads.

**Risks/Dependencies:**

* Kernel/Driver Drift nach OS-Updates.
* NVML-Bindings Stabilität.

**Solution Proposal:**

* NVML-Bindings in Go kapseln; Mock für Tests.
* Warnen statt „silent fail“; klare Reboot-Hinweise.
* NVIDIA Container Toolkit Detection vor Compose-Start.

## Stories

### Story T-009 — GPU-Erkennung & NVML-Probe

**User Story:** Als Nutzer:in möchte ich, dass aistack meine NVIDIA-GPU erkennt und NVML-Funktion prüft, damit GPU-Workloads sicher laufen.

**Scope**

* **In scope:** GPU-Liste, Treiberversion, NVML-Status, JSON-Report.
* **Out of scope:** Treiberinstallation.

**Dependencies & Order**

* Depends on T-001.

**Contracts & Data**

* **Data Model:** `gpu_report.json` ⇒ `{driver_version, nvml_ok, gpus:[{name,uuid,memory_mb}...]}`.

**States & Error Cases**

* NVML nicht verfügbar → `nvml_ok=false`, Hinweis.
* Keine GPU → leere Liste, Warnung.

**Solution Proposal (technical guide)**

* Go-NVML-Bindings kapseln; Mock für Tests.

**Acceptance Criteria (Gherkin-like)**

1. Given RTX 4090 vorhanden, Then `gpu_report.json` enthält `nvml_ok:true` und die GPU-Daten.
2. Given NVML unavailable, Then Report mit `nvml_ok:false` und klarer Log-Warnung.
3. Given keine GPU, Then leere `gpus` und Exit 0.

**Test Plan**

* **Unit:** Parser/Mappings.
* **Integration:** Mocked NVML (ok/fail).

**Migration / Compatibility**

* n/a.

**Feature Flag / Config**

* n/a.

**Risks & Mitigations**

* Treiber-Drift → nur Erkennung, kein Fix.

**Open Questions**

* None.

**Definition of Done (DoD)**

* Report generiert, Logs sauber.

### Story T-010 — NVIDIA Container Toolkit Detection

**User Story:** Als Betreiber:in möchte ich erkennen, ob das NVIDIA-Container-Toolkit korrekt installiert ist, damit GPU an Container durchgereicht wird.

**Scope**

* **In scope:** Runtime-Check für `--gpus` Support, Toolkit-Version.
* **Out of scope:** Installation.

**Dependencies & Order**

* Depends on T-003, T-009.

**Contracts & Data**

* **Events:** `gpu.runtime.check` ⇒ `{docker_support:bool, toolkit_version?:string}`.

**States & Error Cases**

* Toolkit fehlt → klare Handlungsempfehlung.

**Solution Proposal (technical guide)**

* Test-Container mit `--gpus all` trocken starten (dry-run).

**Acceptance Criteria (Gherkin-like)**

1. Given Toolkit vorhanden, Then Event `docker_support:true`.
2. Given fehlt, Then `docker_support:false` und TUI-Hinweis.
3. Given Podman (später), Then Meldung „unsupported in v1“.

**Test Plan**

* **Integration:** Dry-run Ergebnis auswerten.

**Migration / Compatibility**

* n/a.

**Feature Flag / Config**

* n/a.

**Risks & Mitigations**

* Rechteprobleme → aussagekräftige Fehlertexte.

**Open Questions**

* None.

**Definition of Done (DoD)**

* GPU-Durchreichfähigkeit verlässlich erkannt.


# **EP-005 — Metrics & Sensors (GPU/CPU/Temp/Power Estimation)**

**Goal:** Metrik-Collector liefert Auslastung, Temperaturen, Leistungsabschätzung für Entscheidungen und TUI-Anzeige.

**Capabilities:**

* GPU: Util %, VRAM, Temp, Power (NVML).
* CPU: Util %, RAPL-Package-Power (wenn verfügbar).
* Gesamtwatt: Schätzung `gpu_w + cpu_w + baseline_offset`.

**Services/Endpoints:** *n/a*

**Data Contracts:**

* `metrics.sample.jsonl`: `ts, cpu_util, cpu_w, gpu_util, gpu_mem, gpu_w, temp_cpu, temp_gpu, est_total_w`.

**Acceptance (DoD):**

* Given laufender Agent, Then alle 5–10s JSONL-Eintrag in `/var/log/aistack/metrics.log`.
* Given System ohne RAPL, Then Einträge setzen `cpu_w=null` und `est_total_w` nutzt Fallback.

**Risks/Dependencies:**

* Sensorzugriff (hwmon, RAPL) je Hardware.
* Performance Overhead.

**Solution Proposal:**

* Pull-Sampling mit Rate-Limit; P95-Latenz < 50ms/Sample.
* Graceful Degradation ohne RAPL.
* Konfigurierbare Baseline in YAML.

## Stories

### Story T-011 — GPU-Metriken sammeln (Util/VRAM/Temp/Power)

**User Story:** Als Nutzer:in möchte ich periodisch GPU-Metriken, um Auslastung und Verbrauch zu sehen.

**Scope**

* **In scope:** NVML-Sampling alle 10s, JSONL-Log.
* **Out of scope:** CPU/RAPL.

**Dependencies & Order**

* Depends on T-009, T-004.

**Contracts & Data**

* **Data Model:** `metrics.sample.jsonl` ⇒ Felder `ts,gpu_util,gpu_mem,gpu_temp,gpu_w`.

**States & Error Cases**

* NVML transient fail → Eintrag mit `null` Werten + Warnung.
* Sampling-Overhead → Rate-Limit.

**Solution Proposal (technical guide)**

* Agent sammelt, schreibt nach `/var/log/aistack/metrics.log`.

**Acceptance Criteria (Gherkin-like)**

1. Given Agent läuft, Then alle ≤10s ein GPU-Datensatz.
2. Given NVML-Ausfall, Then Logeintrag mit `gpu_* = null`.
3. Given Logrotation, Then neue Datei wird weitergeschrieben.

**Test Plan**

* **Unit:** Formatter/JSONL.
* **Integration:** Mocked NVML ok/fail.

**Migration / Compatibility**

* n/a.

**Feature Flag / Config**

* n/a.

**Risks & Mitigations**

* Logwachstum → Rotation aktiv.

**Open Questions**

* None.

**Definition of Done (DoD)**

* Stabile Metrikerfassung.

### Story T-012 — CPU-Util & RAPL-Leistung erfassen (mit Fallback)

**User Story:** Als Nutzer:in möchte ich CPU-Last und -Leistung erfassen, um Idle-Entscheidungen treffen zu können.

**Scope**

* **In scope:** `/proc/stat` für Util, RAPL (falls verfügbar) für Watt.
* **Out of scope:** Temperatur.

**Dependencies & Order**

* Depends on T-004.

**Contracts & Data**

* **Data Model:** `metrics.sample.jsonl` erweitert um `cpu_util,cpu_w`.

**States & Error Cases**

* Kein RAPL → `cpu_w=null`, Fallback nur Util.

**Solution Proposal (technical guide)**

* Delta-basiert aus `/proc/stat`, RAPL aus `/sys/class/powercap`.

**Acceptance Criteria (Gherkin-like)**

1. Given RAPL verfügbar, Then `cpu_w` enthält numerische Werte.
2. Given RAPL fehlt, Then `cpu_w=null` und keine Fehler.
3. Given hohe CPU-Last, Then `cpu_util` > 80% im Log sichtbar.

**Test Plan**

* **Unit:** Util-Berechnung, RAPL-Parser.
* **Integration:** Simulierte Stat-Dateien.

**Migration / Compatibility**

* n/a.

**Feature Flag / Config**

* n/a.

**Risks & Mitigations**

* Sampling-Jitter → gleitendes Fenster später.

**Open Questions**

* None.

**Definition of Done (DoD)**

* CPU-Metriken im Log vorhanden.


# **EP-006 — Idle Engine & Autosuspend (systemd Integration)**

**Goal:** Konfigurierbare Idle-Erkennung und verlässlicher Suspend-to-RAM mit Inhibit-Logik.

**Capabilities:**

* Gleitendes Fenster für CPU/GPU-Idle.
* Idle-Timer-Visualisierung in TUI.
* `systemd-inhibit` während aktiver Jobs.

**Services/Endpoints:** *n/a*

**Data Contracts:**

* `idle_state.json`: `{ idle_for_s, threshold_s, cpu_idle%, gpu_idle%, gatingReasons:[] }`.

**Acceptance (DoD):**

* Given Idle-Thresholds gesetzt, When Last anliegt, Then kein Suspend.
* Given Idle ≥ Threshold, Then `systemctl suspend` wird ausgelöst und protokolliert.
* Given Re-Start nach Resume, Then Agent läuft weiter und TUI zeigt Timer neu.

**Risks/Dependencies:**

* Fehlinterpretation kurzer Peaks.
* Dienste im Hintergrund (Downloads) vs. „aktiv“.

**Solution Proposal:**

* Hysterese und Minimum-Dauer pro Metrik.
* Gate „HTTP-Traffic niedrig“ (Ports) optional.
* Timer via `aistack-idle.timer` (10s) + stateful Evaluator.

## Stories

### Story T-013 — Idle-Window-Berechnung & State-Ausgabe

**User Story:** Als Nutzer:in möchte ich einen Idle-Status mit Timer sehen, damit ich weiß, wann suspendiert wird.

**Scope**

* **In scope:** Gleitendes Fenster (konfigurierbar), Idle-State JSON.
* **Out of scope:** Suspend auslösen.

**Dependencies & Order**

* Depends on T-011, T-012, T-004.

**Contracts & Data**

* **Data Model:** `idle_state.json` ⇒ `{idle_for_s, threshold_s, cpu_idle_pct, gpu_idle_pct}`.
* **Storage:** `/var/lib/aistack/idle_state.json`.

**States & Error Cases**

* Zu wenig Samples → Idle unbekannt (state `warming_up`).
* Negative Zeiten → abgesichert.

**Solution Proposal (technical guide)**

* Sliding-Window, Hysterese gegen Flapping.

**Acceptance Criteria (Gherkin-like)**

1. Given CPU/GPU niedrig, Then `idle_for_s` steigt monoton.
2. Given Last > Threshold, Then `idle_for_s` reset auf 0.
3. Given Start, Then `warming_up` bis ausreichend Samples.

**Test Plan**

* **Unit:** Window/Hysterese.
* **Integration:** Agent schreibt State-Datei.

**Migration / Compatibility**

* n/a.

**Feature Flag / Config**

* `idle.window_seconds` aus Config.

**Risks & Mitigations**

* Flapping → Hysterese.

**Open Questions**

* None.

**Definition of Done (DoD)**

* Korrekte Idle-State-Ermittlung.

### Story T-014 — Suspend-Executor mit systemd-inhibit-Gate

**User Story:** Als Betreiber:in möchte ich, dass Suspend nur erfolgt, wenn keine aktiven Jobs laufen.

**Scope**

* **In scope:** Gate-Check (Idle ≥ Threshold, keine Inhibits), `systemctl suspend`.
* **Out of scope:** HTTP-Traffic-Gate.

**Dependencies & Order**

* Depends on T-013, T-004.

**Contracts & Data**

* **Events:** `power.suspend.requested|skipped|done` mit Gründen.
* **Guarantees:** Kein Suspend während Inhibit.

**States & Error Cases**

* Inhibit aktiv → Skip mit Grund.
* Suspend-Befehl fehlgeschlagen → Log error.

**Solution Proposal (technical guide)**

* `systemd-inhibit` für eigene kritische Abschnitte.
* Timer ruft Evaluator, der ggf. suspendiert.

**Acceptance Criteria (Gherkin-like)**

1. Given Idle ≥ Threshold, Then `power.suspend.requested` und anschließend `done`.
2. Given Inhibit aktiv, Then `skipped` mit Reason `inhibit`.
3. Given Fehler beim Suspend, Then Error-Log und Exit ≠ 0.

**Test Plan**

* **Unit:** Decision-Gate.
* **Integration:** Dry-run-Modus simuliert Suspend.

**Migration / Compatibility**

* n/a.

**Feature Flag / Config**

* `idle.idle_timeout_seconds`.

**Risks & Mitigations**

* Race-Conditions → Mutex im Agent.

**Open Questions**

* None.

**Definition of Done (DoD)**

* Suspend erfolgt nur wenn erlaubt.


# **EP-007 — Wake-on-LAN Setup & HTTP-Relay (extern)**

**Goal:** OS-seitige WoL-Aktivierung prüfen/setzen und optionalen HTTP→WoL Relay-Container für LAN bereitstellen.

**Capabilities:**

* `ethtool`-basierte WoL-Konfiguration (OS).
* Udev-Rule für Persistenz.
* Externer Mini-Server: `/wake?mac=...&bcast=...&key=...`.

**Services/Endpoints:**

* WoL-Relay: `POST /wake` (LAN only)

**Data Contracts:**

* `wol_config.json`: `{ iface, mac, wol_state }`.
* `relay_request.json`: `{ mac, broadcast, ts, requester }`.

**Acceptance (DoD):**

* Given WoL aktiv, Then `ethtool <iface>` zeigt `Wake-on: g`.
* Given Relay im LAN, When POST `/wake` mit gültigem Key, Then Host wacht auf (verifizierbar via Ping innerhalb N Sekunden).
* Given BIOS/UEFI WoL off, Then TUI zeigt deutlichen Hinweis.

**Risks/Dependencies:**

* BIOS/UEFI Out-of-scope; Nutzer-Action erforderlich.
* Netzwerkswitches, die Broadcast blocken.

**Solution Proposal:**

* Relay als Container-Image (GHCR) liefern (nur Beispiel-Compose).
* Minimaler Access-Key; keine Internet-Exposition.
* TUI-Testknopf „Send Wake Packet“.

## Stories

### Story T-015 — WOL-OS-Konfiguration prüfen & setzen

**User Story:** Als Admin möchte ich WOL am Interface OS-seitig aktivieren, damit der Host aufwachen kann.

**Scope**

* **In scope:** `ethtool`-Check, Set `wol g`, udev-Rule persistieren.
* **Out of scope:** BIOS/UEFI.

**Dependencies & Order**

* Depends on T-003.

**Contracts & Data**

* **Data Model:** `wol_config.json` ⇒ `{iface, mac, wol_state:"g|d"}`.

**States & Error Cases**

* Interface unbekannt → Fehler.
* Rechte fehlen → Fehler.

**Solution Proposal (technical guide)**

* Service `aistack-wol-setup.service` einmalig, udev-Regel in `/etc/udev/rules.d`.

**Acceptance Criteria (Gherkin-like)**

1. Given valides Interface, Then `Wake-on: g` in `ethtool`-Ausgabe.
2. Given Regel aktiv, After Reboot, Then Zustand bleibt `g`.
3. Given ungültiges Interface, Then klare Fehlermeldung.

**Test Plan**

* **Integration:** Parse `ethtool`-Output.
* **Fixtures:** Mock-Ausgaben.

**Migration / Compatibility**

* n/a.

**Feature Flag / Config**

* `wol.interface`, `wol.mac`.

**Risks & Mitigations**

* Switch blockt Broadcast → Hinweis in Doku.

**Open Questions**

* None.

**Definition of Done (DoD)**

* WOL OS-seitig aktiv.

### Story T-016 — HTTP→WOL Relay: Request-Validierung & Magic Packet

**User Story:** Als Betreiber:in möchte ich einen kleinen HTTP-Relay nutzen, der Magic-Packets im LAN sendet.

**Scope**

* **In scope:** Containerisiertes Mini-HTTP, `POST /wake` mit Key, Broadcast-IP.
* **Out of scope:** Deployment auf Zielhost.

**Dependencies & Order**

* None.

**Contracts & Data**

* **API Contracts:**

  | Method | Path    | Request Schema                             | Response Schema | Status/Error Codes | Guarantees                     |
    | ------ | ------- | ------------------------------------------ | --------------- | ------------------ | ------------------------------ |
  | POST   | `/wake` | `{mac:string,broadcast:string,key:string}` | `{status:"ok"}` | 200/400/403/500    | Erfolgreiches UDP-Magic-Packet |

**States & Error Cases**

* Ungültige MAC/Key → 400/403.
* UDP-Error → 500.

**Solution Proposal (technical guide)**

* Kleiner Go-Server, minimaler Access-Key, keine Internet-Exposition.

**Acceptance Criteria (Gherkin-like)**

1. Given gültiger Request, Then 200 und Magic-Packet gesendet.
2. Given falscher Key, Then 403.
3. Given ungültige MAC, Then 400.

**Test Plan**

* **Unit:** MAC/Key-Validator.
* **Integration:** UDP-Send-Mock.

**Migration / Compatibility**

* n/a.

**Feature Flag / Config**

* `wol.relay_url`.

**Risks & Mitigations**

* Missbrauch im LAN → Access-Key Pflicht.

**Open Questions**

* None.

**Definition of Done (DoD)**

* Relay funktionsfähig, testbar.


# **EP-008 — Service: Ollama Orchestration**

**Goal:** Ollama Container installieren, starten, updaten, health-checken und verwalten.

**Capabilities:**

* Lifecycle (install/start/stop/update/remove).
* Port/Volume/Health Management.
* Integration in GPU-Lock.

**Services/Endpoints:**

* Ollama API: `:11434`

**Data Contracts:**

* `service_status.json`: `{ name:"ollama", state:"running|stopped", version, health:"green|yellow|red" }`.

**Acceptance (DoD):**

* Given Install, Then `GET /api/tags` (Ollama) returns 200.
* Given Update verfügbar, Then `aistack update ollama` zieht neues Image und Neustart ist grün.
* Given Remove (keep cache), Then Volume bleibt; bei `--purge` wird es gelöscht.

**Risks/Dependencies:**

* Modelltags/Kompatibilität.
* Netzwerkausfall bei Pull.

**Solution Proposal:**

* HEALTHCHECK in Compose + aktive Probe.
* Versionen optional in `versions.lock`.
* Logs unter `/var/log/aistack/ollama.log` (Proxy/Tail).

## Stories

### Story T-017 — Ollama Lifecycle Commands (install/start/stop/remove)

**User Story:** Als Nutzer:in möchte ich Ollama installieren und verwalten, damit ich den Dienst kontrollieren kann.

**Scope**

* **In scope:** CLI-Befehle, Statusausgabe, Logs-Tail.
* **Out of scope:** Updates (separat).

**Dependencies & Order**

* Depends on T-006.

**Contracts & Data**

* **Events:** `service.ollama.{installed|started|stopped|removed}` mit `{version}`.
* **Data Model:** `service_status.json` ⇒ `{name:"ollama",state,version,health}`.

**States & Error Cases**

* Entfernen bei laufend → erst Stop.
* Volume bleibt erhalten (kein Purge).

**Solution Proposal (technical guide)**

* Wrapper um `docker compose -f compose/ollama.yaml ...`.

**Acceptance Criteria (Gherkin-like)**

1. Given install, Then Status zeigt `running, health:green`.
2. Given stop, Then Port 11434 frei.
3. Given remove, Then Container weg, Volume bleibt.

**Test Plan**

* **Integration:** Start/Stop/Remove Sequenzen.
* **Fixtures:** Mock-Compose.

**Migration / Compatibility**

* n/a.

**Feature Flag / Config**

* n/a.

**Risks & Mitigations**

* Compose-Version drift → pinned.

**Open Questions**

* None.

**Definition of Done (DoD)**

* Lifecycle stabil per CLI/TUI.

### Story T-018 — Ollama Update & Rollback (Service-spezifisch)

**User Story:** Als Nutzer:in möchte ich Ollama updaten und bei Problemen zurückrollen, damit der Service stabil bleibt.

**Scope**

* **In scope:** Pull, Health-Validation, Swap, Rollback.
* **Out of scope:** Global Updater.

**Dependencies & Order**

* Depends on T-017.

**Contracts & Data**

* **Data Model:** `update_plan.json` für Ollama.
* **Guarantees:** Health grün oder Rollback.

**States & Error Cases**

* Netzwerk down → Abbruch, alter Zustand bleibt.
* Health rot → Rollback.

**Solution Proposal (technical guide)**

* Staging-Compose, dann Swap.

**Acceptance Criteria (Gherkin-like)**

1. Given neues Image, Then Update endet mit `health=green`.
2. Given Health rot, Then Rollback aktiv und Service wieder grün.
3. Given Netzfehler, Then kein Zustand geändert.

**Test Plan**

* **Integration:** Simulierter Health-Fail.
* **Fixtures:** Fake-Image-Tags.

**Migration / Compatibility**

* n/a.

**Feature Flag / Config**

* `updates.mode`.

**Risks & Mitigations**

* Tag-Drift → Digest bevorzugen.

**Open Questions**

* None.

**Definition of Done (DoD)**

* Sicheres Update vorhanden.


# **EP-009 — Service: Open WebUI Orchestration**

**Goal:** Open WebUI als UI-Frontend für Ollama verwalten.

**Capabilities:**

* Lifecycle mgmt., Health, Ports, Volumes.
* Konfiguration, welcher Backend-Endpunkt (Ollama/LocalAI) aktuell aktiv ist.

**Services/Endpoints:**

* Open WebUI: `:3000` (lokal/LAN)

**Data Contracts:**

* `ui_binding.json`: `{ active_backend: "ollama|localai", url }`.

**Acceptance (DoD):**

* Given Start, Then HTTP 200 auf `:3000` Health/Root.
* Given Backend Switch, Then UI verbindet stabil zum gewählten Backend.
* Given Stop, Then Port freigegeben und State geloggt.

**Risks/Dependencies:**

* Parallelbetrieb mit LocalAI (GPU-Lock notwendig).
* Config-Drift.

**Solution Proposal:**

* Backend-URL über Env/Config setzen.
* TUI-Schalter „Bind to Ollama/LocalAI“.
* Health-Gating vor Suspend.

## Stories

### Story T-019 — Backend-Switch (Ollama ↔ LocalAI)

**User Story:** Als Nutzer:in möchte ich Open WebUI zwischen Ollama und LocalAI umschalten, damit ich flexibel testen kann.

**Scope**

* **In scope:** TUI-Schalter, Persistenz `ui_binding.json`, Neustart der UI.
* **Out of scope:** GPU-Lock (separat).

**Dependencies & Order**

* Depends on T-007, T-008.

**Contracts & Data**

* **Data Model:** `ui_binding.json` aktualisiert.
* **Events:** `ui.backend.changed` ⇒ `{from,to}`.

**States & Error Cases**

* Ziel-Backend down → Hinweis, trotzdem Schalter gültig.

**Solution Proposal (technical guide)**

* Env-Update + restart.

**Acceptance Criteria (Gherkin-like)**

1. Given Switch to LocalAI, Then UI nutzt neue URL nach Restart.
2. Given Backend down, Then Warnung in TUI, Switch persistiert.
3. Given erneuter Switch, Then State korrekt aktualisiert.

**Test Plan**

* **Integration:** Umschalten mit Dummy-Backends.
* **Fixtures:** Config-Datei.

**Migration / Compatibility**

* n/a.

**Feature Flag / Config**

* n/a.

**Risks & Mitigations**

* Race mit Requests → kurzer Restart-Hinweis.

**Open Questions**

* None.

**Definition of Done (DoD)**

* Umschaltung funktioniert, Zustand gespeichert.


# **EP-010 — Service: LocalAI Orchestration**

**Goal:** LocalAI als alternativer Backend-Service verwalten.

**Capabilities:**

* Lifecycle, Health, Ports, Volumes.
* Modelle getrennt von Ollama managen (Cache-Pfade).

**Services/Endpoints:**

* LocalAI: `:8080`

**Data Contracts:**

* `localai_models.json`: `{ models:[{name,size,updated}] }`.

**Acceptance (DoD):**

* Given Start, Then `/healthz` (oder äquivalent) ist grün.
* Given Model-Download, Then Fortschritt/Fehler in Logs sichtbar.
* Given Remove vs. Purge, Then Verhalten gemäß Einstellung.

**Risks/Dependencies:**

* Modellformate/Quantisierung vs. GPU-Unterstützung.
* Portkollision.

**Solution Proposal:**

* Compose mit Volumes `localai_models`.
* HEALTHCHECK HTTP/Port-Probe.
* Strikte Trennung der Cache-Verzeichnisse.

## Stories

### Story T-020 — LocalAI Lifecycle Commands

**User Story:** Als Nutzer:in möchte ich LocalAI verwalten (install/start/stop/remove), um ein zweites Backend zu haben.

**Scope**

* **In scope:** CLI/TUI-Lifecycle, Status/Logs.
* **Out of scope:** Updates (separat, analog zu Ollama).

**Dependencies & Order**

* Depends on T-008.

**Contracts & Data**

* **Events:** `service.localai.{installed|started|stopped|removed}`.
* **Data Model:** `service_status.json` erweitert.

**States & Error Cases**

* Remove hält `localai_models` (kein Purge).

**Solution Proposal (technical guide)**

* Compose-Wrapper.

**Acceptance Criteria (Gherkin-like)**

1. Given install, Then Health `/healthz` 200.
2. Given stop, Then Port 8080 frei.
3. Given remove, Then Volume bleibt.

**Test Plan**

* **Integration:** Lifecycle-Sequenzen.

**Migration / Compatibility**

* n/a.

**Feature Flag / Config**

* n/a.

**Risks & Mitigations**

* Portkollision → Fehlertext.

**Open Questions**

* None.

**Definition of Done (DoD)**

* Lifecycle stabil.


# **EP-011 — GPU Lock & Concurrency Control**

**Goal:** Exklusives GPU-Mutex verhindert gleichzeitige VRAM-intensive Nutzung durch Open WebUI/LocalAI.

**Capabilities:**

* Datei-/Advisory-Lock + NVML-Sanity.
* TUI-Indikator „GPU locked by X“.

**Services/Endpoints:** *n/a*

**Data Contracts:**

* `gpu_lock.json`: `{ holder:"openwebui|localai|none", since_ts }`.

**Acceptance (DoD):**

* Given LocalAI läuft „heavy“, Then Open WebUI wird blockiert (mit freundlicher Meldung) bis Freigabe.
* Given Lock verloren/Dead, Then Repair räumt Lock sauber auf.

**Risks/Dependencies:**

* Deadlocks durch Crash.
* Falsche Positive bei geringem GPU-Load.

**Solution Proposal:**

* Lock mit Lease/Heartbeat.
* Graceful Backoff/Retry in TUI.
* Repair-Command „force-unlock“ (mit Bestätigung).

## Stories

### Story T-021 — GPU-Mutex (Dateisperre + Lease)

**User Story:** Als Betreiber:in möchte ich einen exklusiven GPU-Lock, damit nicht beide Backends gleichzeitig VRAM-intensiv arbeiten.

**Scope**

* **In scope:** Lock-Datei, Lease/Heartbeat, Holder-Info.
* **Out of scope:** Automatisches Preemption.

**Dependencies & Order**

* Depends on T-009, T-017, T-020.

**Contracts & Data**

* **Data Model:** `gpu_lock.json` ⇒ `{holder, since_ts}`.
* **Events:** `gpu.lock.{acquired|released|stolen}`.

**States & Error Cases**

* Stale Lock → „force-unlock“ mit Bestätigung.
* Crash ohne Release → Lease Timeout.

**Solution Proposal (technical guide)**

* Advisory Lock + Timestamp-Refresh.

**Acceptance Criteria (Gherkin-like)**

1. Given Lock von Open WebUI, Then Start LocalAI blockiert mit Meldung.
2. Given Holder crash, After Lease Timeout, Then Lock freigegeben.
3. Given Force-Unlock, Then Lock entfernt und protokolliert.

**Test Plan**

* **Unit:** Lease-Mechanik.
* **Integration:** Konkurrenzstarts.

**Migration / Compatibility**

* n/a.

**Feature Flag / Config**

* `gpu_lock:true`.

**Risks & Mitigations**

* Deadlocks → Timeout + Force.

**Open Questions**

* None.

**Definition of Done (DoD)**

* Exklusive Nutzung garantiert.


# **EP-012 — Model Management & Caching**

**Goal:** Modelle auswählen, laden, behalten oder gezielt entfernen; Caches standardmäßig persistent.

**Capabilities:**

* TUI-Liste (z. B. Qwen-Familie) + freie Eingabe.
* Keep-Cache bei Uninstall; Purge-Option.
* Speicherübersicht & „evict oldest“.

**Services/Endpoints:** *n/a*

**Data Contracts:**

* `models_state.json`: `{ provider:"ollama|localai", items:[{name, size, path, last_used}] }`.

**Acceptance (DoD):**

* Given Download, Then Fortschritt sichtbar und Abbruch sicher möglich.
* Given Evict oldest, Then Speicher freigegeben und protokolliert.
* Given Purge all, Then alle Modellartefakte entfernt.

**Risks/Dependencies:**

* Große Downloads, instabile Verbindungen.
* Pfadrechte/Quota.

**Solution Proposal:**

* Checksummen/Resume wo möglich.
* Speicherwarnungen ab Schwellwert.
* Einheitliche Cache-Root unter `/var/lib/aistack`.

## Stories

### Story T-022 — Modellliste & Download (Ollama)

**User Story:** Als Nutzer:in möchte ich Modelle in der TUI auswählen und für Ollama herunterladen, damit ich schnell starten kann.

**Scope**

* **In scope:** TUI-Auswahl (inkl. Qwen-Varianten), Fortschritt, Abbruch.
* **Out of scope:** Evict/Purge (separat).

**Dependencies & Order**

* Depends on T-017.

**Contracts & Data**

* **Data Model:** `models_state.json` (`provider:"ollama"`).
* **Events:** `model.download.{started|progress|completed|failed}`.

**States & Error Cases**

* Netzabbrüche → Resume, Retry.
* Speicher knapp → Warnung/Abbruch.

**Solution Proposal (technical guide)**

* Stream-Progress, Resume falls unterstützt.

**Acceptance Criteria (Gherkin-like)**

1. Given Auswahl `qwen2:7b-instruct-q4`, Then Fortschritt sichtbar und Abschluss protokolliert.
2. Given Abbruch, Then Teilartefakte sauber entfernt.
3. Given Netzfehler, Then automatischer Retry (begrenzte Versuche).

**Test Plan**

* **Unit:** Progress/Events.
* **Integration:** Simulierter Netzfehler & Resume.

**Migration / Compatibility**

* n/a.

**Feature Flag / Config**

* `models.keep_cache_on_uninstall:true`.

**Risks & Mitigations**

* Große Files → klare Größenanzeige.

**Open Questions**

* None.

**Definition of Done (DoD)**

* Download nutzbar und robust.

### Story T-023 — Cache-Übersicht & Evict Oldest (Ollama/LocalAI)

**User Story:** Als Nutzer:in möchte ich belegten Speicher sehen und gezielt älteste Modelle entfernen.

**Scope**

* **In scope:** Größenberechnung, Last-Used, „Evict oldest“.
* **Out of scope:** Vollständiger Purge.

**Dependencies & Order**

* Depends on T-022, T-020.

**Contracts & Data**

* **Data Model:** `models_state.json` erweitert um `{size,last_used}`.

**States & Error Cases**

* Fehlende Metadaten → Fallback auf Dateialter.

**Solution Proposal (technical guide)**

* FS-Scan + Metadaten-Index.

**Acceptance Criteria (Gherkin-like)**

1. Given Cache > X GB, Then Warnung in TUI.
2. Given Evict oldest, Then Speicher reduziert und Log-Eintrag erstellt.
3. Given Metadaten fehlen, Then Fallback funktioniert.

**Test Plan**

* **Unit:** Sortierlogik.
* **Integration:** FS-Setup mit Dummy-Dateien.

**Migration / Compatibility**

* n/a.

**Feature Flag / Config**

* n/a.

**Risks & Mitigations**

* Symlinks → realpath prüfen.

**Open Questions**

* None.

**Definition of Done (DoD)**

* Speicherverwaltung sichtbar & steuerbar.


# **EP-013 — TUI/CLI UX (Profiles, Navigation, Logs)**

**Goal:** Mausfreie, farbige TUI mit Profilen, Nummern-/Pfeiltasten, Log-Viewer und klaren Fehlermeldungen.

**Capabilities:**

* Profile: Minimal, Standard-GPU (Default, ohne Auto-Modelle), Dev.
* Nummern/Enter/Space, fzf-ähnliche Listen.
* Live-Status, Idle-Timer, Logs tailen.

**Services/Endpoints:** *n/a*

**Data Contracts:**

* `ui_state.json`: `{ menu, selection, last_error }`.

**Acceptance (DoD):**

* Given Tastatur only, Then alle Kernaktionen sind erreichbar.
* Given Fehlerzustand, Then TUI zeigt Ursache + „how to fix“ Link.
* Given Log-Viewer, Then Tail von Service-Logs funktioniert.

**Risks/Dependencies:**

* Terminal-Fähigkeiten (ANSI).
* Internationalisierung (Englisch-only in v1 gewünscht).

**Solution Proposal:**

* Konsistente Hotkeys, Hilfe-Overlay `?`.
* JSON-Logs, kurzer human-String im UI.
* Farbschema dezent, hoher Kontrast.

## Stories

### Story T-024 — Hauptmenü & Navigation (Nummern/Pfeile/Enter/Space)

**User Story:** Als Nutzer:in möchte ich per Tastatur durch Hauptmenüs navigieren, um Aktionen ohne Maus auszuführen.

**Scope**

* **In scope:** Menüpunkte: Status, Install/Uninstall, Models, Power, Logs, Diagnostics, Settings, Update.
* **Out of scope:** Screen-Details (separat).

**Dependencies & Order**

* Depends on T-002.

**Contracts & Data**

* **Data Model:** `ui_state.json` ⇒ `{menu, selection, last_error}`.

**States & Error Cases**

* Falsche Eingabe → Fehlermeldung oben sichtbar.

**Solution Proposal (technical guide)**

* Keymap: Zahlen = Direktwahl, Pfeile = Fokus, Space = select.

**Acceptance Criteria (Gherkin-like)**

1. Given TUI offen, Then alle Menüs per Nummer erreichbar.
2. Given Pfeile/Enter, Then Fokus/Bestätigung funktionieren.
3. Given Fehler, Then Meldung im Statusbereich.

**Test Plan**

* **Unit:** Keybindings.
* **Integration:** Snapshot-Tests der Views.

**Migration / Compatibility**

* n/a.

**Feature Flag / Config**

* n/a.

**Risks & Mitigations**

* Terminalgröße klein → responsives Layout.

**Open Questions**

* None.

**Definition of Done (DoD)**

* Bedienung flüssig, keine Maus nötig.


# **EP-014 — Health Checks & Repair Flows**

**Goal:** Einheitliche Health-Evaluierung und automatisierte Reparaturen (idempotent) für Services und Units.

**Capabilities:**

* HTTP/Port-Probes, GPU-Smoke-Test.
* `repair` setzt Container/Units neu auf.
* Drift-Detection (Version, Ports, Volumes).

**Services/Endpoints:** *n/a*

**Data Contracts:**

* `health_report.json`: `{ services:[{name,health,msg}], gpu:{ok,msg} }`.

**Acceptance (DoD):**

* Given defekter Service, When `aistack repair --service X`, Then Zustand grün ohne Datenverlust (außer Purge).
* Given Version-Drift, Then Update/Repair korrigiert drift deterministisch.

**Risks/Dependencies:**

* Falsch-negative Health-Probes.
* Partial-Failures (Netzwerk zeitweise down).

**Solution Proposal:**

* Mehrstufig: Schnellprobe → Deepcheck → Remediation.
* Dry-Run-Option mit Plan-Ausgabe.
* Retry-Backoff, Circuit-Breaker.

## Stories

### Story T-025 — Health-Reporter (Services + GPU Smoke)

**User Story:** Als Betreiber:in möchte ich einen einheitlichen Health-Report sehen, um Störungen schnell zu erkennen.

**Scope**

* **In scope:** HTTP/Port-Probes, GPU-Schnelltest, aggregierter Report.
* **Out of scope:** Repair-Actions.

**Dependencies & Order**

* Depends on T-006, T-007, T-008, T-009.

**Contracts & Data**

* **Data Model:** `health_report.json` ⇒ `{services:[{name,health,msg}],gpu:{ok,msg}}`.

**States & Error Cases**

* Teilweise Ausfälle → `yellow` mit Nachricht.

**Solution Proposal (technical guide)**

* Mehrstufig: Port → HTTP → GPU-Minicall.

**Acceptance Criteria (Gherkin-like)**

1. Given alle Services laufen, Then Report zeigt alle `green`.
2. Given ein Service down, Then `red` mit Fehlertext.
3. Given NVML fail, Then `gpu.ok=false` und Hinweis.

**Test Plan**

* **Unit:** Aggregationslogik.
* **Integration:** Dienste gezielt stoppen.

**Migration / Compatibility**

* n/a.

**Feature Flag / Config**

* n/a.

**Risks & Mitigations**

* Flaky Checks → Retry/Backoff.

**Open Questions**

* None.

**Definition of Done (DoD)**

* Konsistenter Health-Report.

### Story T-026 — Repair-Command für einzelne Services

**User Story:** Als Nutzer:in möchte ich einen `repair`-Befehl pro Service, um defekte Deployments automatisch zu heilen.

**Scope**

* **In scope:** Stop → Remove → Recreate (ohne Volume-Löschung), Health-Recheck.
* **Out of scope:** Purge.

**Dependencies & Order**

* Depends on T-025.

**Contracts & Data**

* **Events:** `service.X.repair.{started|completed|failed}`.

**States & Error Cases**

* Health weiter rot → `failed` mit Details.

**Solution Proposal (technical guide)**

* Deterministische Reihenfolge, klare Logs.

**Acceptance Criteria (Gherkin-like)**

1. Given defekter Service, When repair, Then Health grün.
2. Given weiterhin Fehler, Then `failed` mit Ursache.
3. Given intakter Service, Then repair no-op (Exit 0).

**Test Plan**

* **Integration:** Defekt simulieren (falsche Env), Repair testen.

**Migration / Compatibility**

* n/a.

**Feature Flag / Config**

* n/a.

**Risks & Mitigations**

* Datenverlust vermeiden → Volumes unberührt.

**Open Questions**

* None.

**Definition of Done (DoD)**

* Repair zuverlässig.


# **EP-015 — Logging, Diagnostics & Diff-friendly Reports**

**Goal:** Strukturierte Logs, Diagnosepakete (ZIP) und Systemzustandsberichte für Support & Vergleich.

**Capabilities:**

* JSON-Logs unter `/var/log/aistack`.
* `aistack diag --out file.zip` (Logs, Config, Health, Versions).
* Redaction sensibler Daten.

**Services/Endpoints:** *n/a*

**Data Contracts:**

* `diag_manifest.json`: `{ files:[], created_ts, host }`.
* `config_snapshot.yaml` (redacted).

**Acceptance (DoD):**

* Given `aistack diag`, Then ZIP enthält vollständige, redacted Artefakte und verifizierbare Checksums.
* Given großer Logumfang, Then Rotation greift ohne Datenverlust (letzte N MB behalten).

**Risks/Dependencies:**

* Geheimnisse in Configs.
* Logwachstum.

**Solution Proposal:**

* Logrotate (size-basiert), maximale Aufbewahrung.
* Secrets-Redaction-Filter mit Unit-Tests.
* Zeitstempel ISO-8601, Host-Metadaten minimal.

## Stories

### Story T-027 — Strukturierte JSON-Logs & Rotation

**User Story:** Als Betreiber:in möchte ich strukturierte Logs unter `/var/log/aistack`, damit Diagnose einfach ist.

**Scope**

* **In scope:** JSON-Formatter, Log-Level, logrotate-Regel.
* **Out of scope:** Diag-ZIP.

**Dependencies & Order**

* Depends on T-004.

**Contracts & Data**

* **Storage:** `/var/log/aistack/*.log` (max size, N files).

**States & Error Cases**

* Schreibrechte fehlen → Fehlermeldung und Fallback `stderr`.

**Solution Proposal (technical guide)**

* Level: debug/info/warn/error; Format: ISO-8601.

**Acceptance Criteria (Gherkin-like)**

1. Given Aktionen, Then Logs erscheinen als JSON in Datei.
2. Given Rotation, Then neue Datei genutzt ohne Verlust.
3. Given Level `warn`, Then `info` wird gefiltert.

**Test Plan**

* **Unit:** Logger-Konfiguration.
* **Integration:** Rotation-Force.

**Migration / Compatibility**

* n/a.

**Feature Flag / Config**

* `logging.level`, `logging.format: json`.

**Risks & Mitigations**

* Disk-Füllung → Größe begrenzen.

**Open Questions**

* None.

**Definition of Done (DoD)**

* Logs konsistent & rotiert.

### Story T-028 — Diagnosepaket (ZIP) mit Redaction

**User Story:** Als Nutzer:in möchte ich ein Diagnosepaket erzeugen, damit ich Fehler nachträglich analysieren kann.

**Scope**

* **In scope:** Sammeln Logs, Config-Snapshot (redacted), Health, Versions.
* **Out of scope:** Live-Upload.

**Dependencies & Order**

* Depends on T-025, T-027.

**Contracts & Data**

* **Data Model:** `diag_manifest.json` ⇒ `{files:[],created_ts,host}`.

**States & Error Cases**

* Redaction versagt → Abort mit Warnung.

**Solution Proposal (technical guide)**

* ZIP erzeugen, Checksums je Datei.

**Acceptance Criteria (Gherkin-like)**

1. Given `aistack diag --out file.zip`, Then ZIP enthält Manifest + Artefakte.
2. Given Secrets in Config, Then redacted im Snapshot.
3. Given fehlende Logs, Then Manifest listet sie als „missing“.

**Test Plan**

* **Unit:** Redaction-Filter.
* **Integration:** ZIP-Inhalt prüfen.

**Migration / Compatibility**

* n/a.

**Feature Flag / Config**

* n/a.

**Risks & Mitigations**

* PII-Leak → strikte Redaction-Regeln.

**Open Questions**

* None.

**Definition of Done (DoD)**

* Diagnosepaket reproduzierbar.


# **EP-016 — Update & Rollback (Binary & Containers)**

**Goal:** Sichere Updates der Binary und Container-Images mit optionalem Rollback.

**Capabilities:**

* Check auf neue Releases/Tags (GitHub).
* Atomic Swap: erst Pull & Validate, dann Switch.
* Rollback bei Health-Fail.

**Services/Endpoints:** *n/a*

**Data Contracts:**

* `update_plan.json`: `{ current, target, steps:[...] }`.
* `versions.lock` (optional, pro Service).

**Acceptance (DoD):**

* Given neues Image, Then Update führt zu `health=green` oder Rollback auf vorherige Version.
* Given Selbst-Update aktiviert, Then Signatur/Checksumme validiert.

**Risks/Dependencies:**

* Netzabbrüche beim Pull.
* Selbst-Update fehlschlägt (Permission).

**Solution Proposal:**

* Validate in Staging-Compose (isoliert) vor Swap.
* Checksums/Signaturen; Cosign optional.
* Reentrante Updates mit Journal-Eintrag.

## Stories

### Story T-029 — Container-Update „all“ mit Health-Gate

**User Story:** Als Nutzer:in möchte ich alle Service-Images aktualisieren können, ohne Downtime-Risiko.

**Scope**

* **In scope:** Pull, Health-Check, sequentieller Swap, Rollback.
* **Out of scope:** Binary-Self-Update.

**Dependencies & Order**

* Depends on T-018, T-020.

**Contracts & Data**

* **Data Model:** `update_plan.json` ⇒ `steps, current, target`.

**States & Error Cases**

* Ein Service fail → nur dieser rollback.

**Solution Proposal (technical guide)**

* Reihenfolge: LocalAI → Ollama → Open WebUI.

**Acceptance Criteria (Gherkin-like)**

1. Given Updates vorhanden, Then alle Services nacheinander grün.
2. Given ein Health-Fail, Then nur betroffener Service rollback.
3. Given keine Updates, Then no-op (Exit 0).

**Test Plan**

* **Integration:** Mix aus good/bad Images.
* **Coverage:** Erfolg/Teil-Fail.

**Migration / Compatibility**

* n/a.

**Feature Flag / Config**

* `updates.mode`.

**Risks & Mitigations**

* Tag-Drift → Digests nutzen.

**Open Questions**

* None.

**Definition of Done (DoD)**

* Sicheres Multi-Update vorhanden.


# **EP-017 — Security, Permissions & Secrets**

**Goal:** Minimale Rechte, klare Besitzverhältnisse und sichere lokale Geheimnisspeicherung.

**Capabilities:**

* Gruppe/Nutzer `aistack`, Besitz `/var/lib/aistack`.
* Secrets verschlüsselt (libsodium secretbox) + Passphrase-Datei.
* Keine Telemetrie, lokale Logs only.

**Services/Endpoints:** *n/a*

**Data Contracts:**

* `secrets_index.json`: `{ entries:[{name, last_rotated}] }`.

**Acceptance (DoD):**

* Given Standardbetrieb, Then Services laufen nicht als root (wo möglich).
* Given Secret-Write, Then Datei verschlüsselt, Rechte 600.

**Risks/Dependencies:**

* Headless Keyring fehlt → Datei-gestützte Passphrase nötig.
* Usability vs. Sicherheit.

**Solution Proposal:**

* `chmod 600`, Besitzer `aistack`.
* Passphrase-Management mit klarer Recovery-Anleitung.
* Sudo-Aktionen explizit, auditierbar.

## Stories

### Story T-030 — Lokale Secret-Verschlüsselung (libsodium)

**User Story:** Als Nutzer:in möchte ich Secrets lokal verschlüsselt speichern, damit nichts im Klartext liegt.

**Scope**

* **In scope:** Secretbox-Encryption, Passphrase-Datei, Rechte 600.
* **Out of scope:** Key-Rotation.

**Dependencies & Order**

* Depends on T-001.

**Contracts & Data**

* **Data Model:** `secrets_index.json` ⇒ `{entries:[{name,last_rotated}]}`.
* **Storage:** `/var/lib/aistack/secrets/*`.

**States & Error Cases**

* Fehlende Passphrase → Fehler + Anleitung.
* Falsche Rechte → Fix & Warnung.

**Solution Proposal (technical guide)**

* libsodium Wrapper, Passphrase in Root-Only-Datei.

**Acceptance Criteria (Gherkin-like)**

1. Given Secret gespeichert, Then Datei verschlüsselt und Rechte 600.
2. Given Passphrase fehlt, Then Fehlermeldung mit Pfad.
3. Given Read, Then korrekter Klartext geliefert.

**Test Plan**

* **Unit:** Encrypt/Decrypt, Rechte-Check.
* **Integration:** Fehlende Passphrase.

**Migration / Compatibility**

* n/a.

**Feature Flag / Config**

* n/a.

**Risks & Mitigations**

* Headless UX → klare Doku.

**Open Questions**

* None.

**Definition of Done (DoD)**

* Secrets sicher gespeichert.


# **EP-018 — Configuration Management (YAML)**

**Goal:** Konsolidierte Konfiguration mit systemweiter und Nutzer-spezifischer YAML, inkl. Profile.

**Capabilities:**

* `/etc/aistack/config.yaml` (system), `~/.aistack/config.yaml` (user).
* Merge-Strategie system→user.
* Validierung und Defaulting.

**Services/Endpoints:** *n/a*

**Data Contracts:**

* `config.schema` (verbale Kurzbeschreibung):

    * `container_runtime`, `profile`, `gpu_lock`
    * `idle.{cpu_idle_threshold,gpu_idle_threshold,window_seconds,idle_timeout_seconds}`
    * `power_estimation.baseline_watts`
    * `wol.{interface,mac,relay_url}`
    * `logging.{level,format}`
    * `models.keep_cache_on_uninstall`
    * `updates.mode`

**Acceptance (DoD):**

* Given fehlende Felder, Then Defaults greifen dokumentiert.
* Given invalide Werte, Then TUI zeigt Validierungsfehler mit Pfad.

**Risks/Dependencies:**

* YAML-Parsing-Fallen (Tabs/Spaces).
* Konfig-Drift.

**Solution Proposal:**

* Strikte Schema-Validierung, Pfad-basierte Fehlermeldungen.
* `aistack config test` Befehl.
* Snapshot in Diag-Paket.

## Stories

### Story T-031 — Config-Parsing & Defaulting (System + User Merge)

**User Story:** Als Nutzer:in möchte ich eine YAML-Konfig mit Defaults und Merge von system/user, damit ich Einstellungen zentral steuern kann.

**Scope**

* **In scope:** `/etc/aistack/config.yaml` + `~/.aistack/config.yaml`, Schema-Validierung, Defaults.
* **Out of scope:** Live-Reload.

**Dependencies & Order**

* Depends on T-001.

**Contracts & Data**

* **Data Model:** `config.schema` (verbale Spezifikation).
* **Events:** `config.validation.{ok|error}`.

**States & Error Cases**

* Ungültige Felder → Validierungsfehler mit Pfad.

**Solution Proposal (technical guide)**

* Merge system→user, strikte Typprüfung.

**Acceptance Criteria (Gherkin-like)**

1. Given fehlende Felder, Then Defaults greifen dokumentiert.
2. Given invalide Werte, Then Fehler mit YAML-Pfad.
3. Given `aistack config test`, Then Exit 0/≠0 je nach Zustand.

**Test Plan**

* **Unit:** Parser/Defaults/Validation.
* **Integration:** Beispielconfigs.

**Migration / Compatibility**

* n/a.

**Feature Flag / Config**

* Alle genannten Felder (idle, wol, logging, updates, models, runtime).

**Risks & Mitigations**

* Tabs in YAML → Hinweis & Beispiel.

**Open Questions**

* None.

**Definition of Done (DoD)**

* Config robust & validierbar.


# **EP-019 — CI/CD (GitHub Actions) & Teststrategie**

**Goal:** Vollständige Pipeline mit Lint, Unit, Integration, E2E, Release-Artifakten und Qualitätsgates.

**Capabilities:**

* Build-Matrix linux/amd64.
* Docker-in-Docker für Integration.
* E2E-VM-Job (ohne echte GPU; NVML gemockt).

**Services/Endpoints:** *n/a*

**Data Contracts:**

* `ci_report.json`: `{ job, status, coverage }`.

**Acceptance (DoD):**

* Given Pull Request, Then Pipeline blockiert Merge bei Lint/Test-Fails.
* Given Release-Tag, Then Binary + Checksums + Changelog veröffentlicht.
* Given Flakes, Then Quarantäne-Label für instabile Tests.

**Risks/Dependencies:**

* NVML-Mocks realistisch halten.
* Selbstgehostete Runner optional.

**Solution Proposal:**

* golangci-lint, race-detector aktiv.
* Coverage-Ziel ≥ 80% Kernpakete.
* Keep-a-Changelog Format, semver Tags.

## Stories

### Story T-032 — CI: Lint + Unit Tests + Artefakt-Upload (Snapshot)

**User Story:** Als Maintainer möchte ich, dass PRs automatisch gelintet und getestet werden und ein Snapshot-Build entsteht.

**Scope**

* **In scope:** Workflow mit Lint/Test, Coverage-Gate, Artefakt `aistack` (linux/amd64).
* **Out of scope:** Release-Tagging, E2E.

**Dependencies & Order**

* Depends on T-001.

**Contracts & Data**

* **Data Model:** `ci_report.json` ⇒ `{job,status,coverage}`.

**States & Error Cases**

* Coverage < Ziel → Build rot.

**Solution Proposal (technical guide)**

* `golangci-lint`, `go test -race -cover`, Upload artefact.

**Acceptance Criteria (Gherkin-like)**

1. Given PR geöffnet, Then Lint/Test laufen und Gate bei Fail blockt.
2. Given Tests grün, Then Artefakt verfügbar.
3. Given Coverage < 80% Kernpakete, Then Fail.

**Test Plan**

* **Meta:** Workflow selbst verifiziert durch Run-Status.

**Migration / Compatibility**

* n/a.

**Feature Flag / Config**

* n/a.

**Risks & Mitigations**

* Flaky Tests → Quarantäne (später).

**Open Questions**

* None.

**Definition of Done (DoD)**

* CI-Grundpipeline aktiv.


# **EP-020 — Uninstall & Purge (Idempotent Destroy)**

**Goal:** Vollständige Deinstallation mit optionalem Purge (alles inkl. Modelle & Configs entfernen).

**Capabilities:**

* `aistack uninstall --service X` (keep caches).
* `aistack purge --all` (harte Entfernung).
* Doppelbestätigung für destructive Actions.

**Services/Endpoints:** *n/a*

**Data Contracts:**

* `uninstall_log.json`: `{ target, keep_cache, removed_items:[...] }`.

**Acceptance (DoD):**

* Given Uninstall, Then Dienste/Units/Netzwerk entfernt, Modelle optional behalten.
* Given Purge, Then `/var/lib/aistack` leer und Units weg.
* Given erneuter Bootstrap, Then Neuinstallation ohne Restartefakte möglich.

**Risks/Dependencies:**

* Zurückbleibende Docker-Netze/Volumes.
* Rechteprobleme beim Löschen.

**Solution Proposal:**

* Orchestrierte Reihenfolge: stop → rm → volumes/net → files.
* Post-Verify: „no leftovers“ Prüfung.
* Report im Diag-Paket.

## Stories

### Story T-033 — Uninstall pro Service (Keep Caches)

**User Story:** Als Nutzer:in möchte ich einzelne Services deinstallieren, ohne Modell-Caches zu löschen.

**Scope**

* **In scope:** Stop/Remove, Volumes behalten, Post-Verify.
* **Out of scope:** Purge.

**Dependencies & Order**

* Depends on T-017, T-020.

**Contracts & Data**

* **Data Model:** `uninstall_log.json` ⇒ `{target,keep_cache:true,removed_items:[...]}`.

**States & Error Cases**

* Läuft noch → zuerst Stop.

**Solution Proposal (technical guide)**

* Compose down ohne `-v`.

**Acceptance Criteria (Gherkin-like)**

1. Given uninstall, Then Container/Netzwerk weg, Volumes intakt.
2. Given erneut uninstall, Then no-op Exit 0.
3. Given Fehler, Then detaillog mit to-be-fixed.

**Test Plan**

* **Integration:** Install → Uninstall → Reinstall.

**Migration / Compatibility**

* n/a.

**Feature Flag / Config**

* n/a.

**Risks & Mitigations**

* Reste → Verify nach Abschluss.

**Open Questions**

* None.

**Definition of Done (DoD)**

* Deinstallation rückstandsfrei (ohne Caches).

### Story T-034 — Purge All (inkl. Modelle & Configs) mit Doppel-Confirm

**User Story:** Als Nutzer:in möchte ich alles vollständig entfernen können, inklusive Modelle und Configs.

**Scope**

* **In scope:** Stop/Remove aller Services, Volumes/Libraries/Configs löschen, Doppelbestätigung.
* **Out of scope:** Reinstall.

**Dependencies & Order**

* Depends on T-033.

**Contracts & Data**

* **Events:** `purge.started|completed`.
* **Guarantees:** Danach keine Artefakte in `/var/lib/aistack`, `/etc/aistack`.

**States & Error Cases**

* Rechteprobleme → Fehler mit Liste verbleibender Pfade.

**Solution Proposal (technical guide)**

* Reihenfolge: Services → Netz/Volumes → Files → Verify.

**Acceptance Criteria (Gherkin-like)**

1. Given purge, Then alle Artefakte entfernt und Verify zeigt empty.
2. Given fehlende Rechte, Then Liste verbleibender Pfade in Report.
3. Given anschließend Bootstrap, Then Neuinstallation fehlerfrei.

**Test Plan**

* **Integration:** Full cycle install→purge→install.

**Migration / Compatibility**

* n/a.

**Feature Flag / Config**

* n/a.

**Risks & Mitigations**

* Datenverlust → Doppel-Confirm.

**Open Questions**

* None.

**Definition of Done (DoD)**

* Purge vollständig & verifiziert.


# **EP-021 — Update Policy & Version Locking**

**Goal:** Rolling-Standard mit optionalem Lockfile pro Service, geprüft und dokumentiert.

**Capabilities:**

* `versions.lock` erkennen/erzwingen.
* Policy „rolling“ vs. „pinned“.

**Services/Endpoints:** *n/a*

**Data Contracts:**

* `versions.lock` Kurzformat: `service:image@tag|digest`.

**Acceptance (DoD):**

* Given `rolling`, Then `aistack update all` zieht neueste stabile Tags.
* Given Lockfile, Then nur definierte Versionen werden verwendet.

**Risks/Dependencies:**

* Tag-Drift; „latest“ Anti-Pattern.
* Digest-Verfügbarkeit.

**Solution Proposal:**

* Prefer digests für deterministische Deploys.
* Lockfile-Migration und Lint.

## Stories

### Story T-035 — Versions-Lockfile lesen/erzwingen

**User Story:** Als Betreiber:in möchte ich ein Lockfile nutzen, damit Services deterministische Versionen verwenden.

**Scope**

* **In scope:** `versions.lock` Parser, Enforcement beim Start/Update.
* **Out of scope:** Lockfile-Generator.

**Dependencies & Order**

* Depends on T-006–T-008.

**Contracts & Data**

* **Data Model:** `versions.lock` ⇒ `service:image@tag|digest`.

**States & Error Cases**

* Ungültiger Eintrag → Start verweigert mit Fehler.

**Solution Proposal (technical guide)**

* Digests bevorzugen; Fallback Tag.

**Acceptance Criteria (Gherkin-like)**

1. Given gültiges Lock, Then Start nutzt exakt definierte Images.
2. Given ungültiges Lock, Then klarer Fehler mit Zeilennummer.
3. Given `updates.mode=pinned`, Then Updates blockiert.

**Test Plan**

* **Unit:** Parser/Validator.
* **Integration:** Start mit Lock, Update-Block.

**Migration / Compatibility**

* n/a.

**Feature Flag / Config**

* `updates.mode: pinned|rolling`.

**Risks & Mitigations**

* Divergente Tags → Digests erzwingen.

**Open Questions**

* None.

**Definition of Done (DoD)**

* Locking wirkt deterministisch.


# **EP-022 — Documentation & Ops Playbooks**

**Goal:** Prägnante Doku (Englisch, nerdy v0.1) für Setup, Betrieb, Troubleshooting, Power & WoL.

**Capabilities:**

* README (Quickstart), OPERATIONS (Playbooks), POWER & WOL Guide.
* Fehlerkatalog mit „how to fix“.

**Services/Endpoints:** *n/a*

**Data Contracts:**

* `docs/*.md` (strukturierte Abschnitte; keine Secrets).

**Acceptance (DoD):**

* Given Anfänger:in, Then Quickstart führt reproduzierbar zu „services green“ ≤ 10 Minuten (ohne Modelle).
* Given WoL-Probleme, Then Playbook ermöglicht Diagnose bis zum grünen Zustand.

**Risks/Dependencies:**

* Doku veraltet; CI sollte Link-Checks machen.

**Solution Proposal:**

* Docs in CI link-checken.
* Beispiele für `config.yaml` & `versions.lock`.
* TUI-Screenshots (ASCII).

---

### Assumptions (konservativ, pro Epic gruppiert)

* EP-003/008–010: Podman-Unterstützung nur „best effort“; Docker ist Default.
* EP-004/005: Exakte Gesamtwattmessung ohne externe Hardware nicht möglich; Schätzung via GPU/CPU + Baseline.
* EP-007: BIOS/UEFI-WoL muss manuell aktiviert sein.
* EP-011/012: Gleichzeitiger Betrieb der UIs möglich, aber GPU-Lock verhindert Heavy-Overlap.

## Stories

### Story T-036 — README Quickstart (Bootstrap → Services green ≤10 min)

**User Story:** Als Nutzer:in möchte ich eine kurze Anleitung, um das System schnell lauffähig zu bekommen.

**Scope**

* **In scope:** Voraussetzungen, Einzeiler-Install, „services green“ Checkliste.
* **Out of scope:** Tiefen-Doku.

**Dependencies & Order**

* Depends on T-003, T-006–T-008.

**Contracts & Data**

* **Data Model:** `docs/README.md` (Inhalt als Dokument, kein Code hier im Ticket).

**States & Error Cases**

* Veraltete Hinweise → CI Linkcheck (später).

**Solution Proposal (technical guide)**

* Schrittfolge: Bootstrap → Install profile → Health verify.

**Acceptance Criteria (Gherkin-like)**

1. Given frische Ubuntu-VM, Then Schritte führen zu grünen Services ≤10 min (ohne Modelle).
2. Given Befehle copy/paste, Then keine Fehlersyntax.
3. Given Troubleshooting-Abschnitt, Then häufige Fehler abgedeckt.

**Test Plan**

* **E2E:** Manuelle Durchlaufprobe in frischer VM (ohne GPU).

**Migration / Compatibility**

* n/a.

**Feature Flag / Config**

* n/a.

**Risks & Mitigations**

* Drift → regelmäßige Pflege.

**Open Questions**

* None.

**Definition of Done (DoD)**

* README pragmatisch & korrekt.

---
