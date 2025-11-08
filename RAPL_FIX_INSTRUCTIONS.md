# RAPL Permission Fix - Anleitung für Ubuntu Server

## Problem
Die RAPL CPU Power Monitoring funktioniert nicht wegen Permission Denied Fehler:
```
{"level":"warn","type":"cpu.rapl.read.failed","payload":{"error":"permission denied"}}
```

## Lösung
Ich habe das Problem mit zwei Methoden behoben:
1. **udev rule** - korrigiert (ACTION-Filter entfernt)
2. **systemd-tmpfiles** - NEU hinzugefügt (zuverlässiger für sysfs Dateien)

## Installation (3 Optionen)

### Option 1: Vollständige Reinstallation (Empfohlen)
```bash
cd ~/aistack
git pull
sudo ./install.sh
```

Das installiert:
- ✅ Korrigierte udev rule
- ✅ Neue tmpfiles.d Konfiguration
- ✅ Wendet Permissions sofort an

### Option 2: Nur tmpfiles deployen (Schneller)
```bash
cd ~/aistack
git pull

# Deploye tmpfiles Konfiguration
sudo cp assets/tmpfiles.d/aistack-rapl.conf /etc/tmpfiles.d/

# Wende sofort an
sudo systemd-tmpfiles --create /etc/tmpfiles.d/aistack-rapl.conf

# Prüfe Permissions (sollte jetzt 644 sein)
ls -la /sys/class/powercap/intel-rapl/intel-rapl:0/energy_uj

# Restart Agent
sudo systemctl restart aistack-agent
```

### Option 3: Manuelle Permissions (Temporär)
Nur bis zum nächsten Reboot gültig:
```bash
sudo chmod 644 /sys/class/powercap/intel-rapl/*/energy_uj
sudo chmod 644 /sys/class/powercap/intel-rapl/*/*/energy_uj
sudo systemctl restart aistack-agent
```

## Verifikation

Nach dem Fix sollte RAPL funktionieren:

```bash
# Logs prüfen
sudo journalctl -u aistack-agent -f | grep rapl
```

**Erfolg** sieht so aus:
```json
{"level":"info","type":"cpu.rapl.detected","message":"RAPL power monitoring available","payload":{"path":"/sys/class/powercap/intel-rapl/intel-rapl:0/energy_uj"}}
```

**Fehler** sieht so aus (sollte nicht mehr erscheinen):
```json
{"level":"warn","type":"cpu.rapl.read.failed","message":"Failed to read RAPL","payload":{"error":"permission denied"}}
```

## Technische Details

### Warum funktionierte die alte udev rule nicht?

Die alte udev rule hatte:
```
SUBSYSTEM=="powercap", KERNEL=="intel-rapl:*", ACTION=="add|change", MODE="0644"
```

**Problem**: `ACTION=="add|change"` triggert nur wenn Geräte hinzugefügt/geändert werden.
RAPL sysfs Dateien existieren bereits beim Boot, daher wird die Rule nie ausgeführt.

**Fix**: ACTION-Filter entfernt:
```
SUBSYSTEM=="powercap", KERNEL=="intel-rapl:*", MODE="0644"
```

### Warum systemd-tmpfiles zusätzlich?

`systemd-tmpfiles` ist die empfohlene Methode für sysfs Permission Management:
- Wird bei jedem Boot ausgeführt
- Kann manuell getriggert werden (`--create`)
- Zuverlässiger als udev für bereits existierende Dateien

Die tmpfiles Konfiguration (`/etc/tmpfiles.d/aistack-rapl.conf`):
```
z /sys/class/powercap/intel-rapl/intel-rapl:*/energy_uj 0644 - - -
z /sys/class/powercap/intel-rapl/intel-rapl:*/intel-rapl:*/energy_uj 0644 - - -
```

`z` = Set file permissions and ownership (don't follow symlinks)

## Was wurde noch hinzugefügt?

- ✅ `docs/TROUBLESHOOTING.md` - Umfassendes Troubleshooting Guide
  - RAPL Permission Fix (3 Optionen)
  - Idle State Reset Problem (Diagnose & Lösungen)
  - GPU Lock Issues
  - Häufige Probleme

## Commits

1. `fix: RAPL permissions with tmpfiles.d and troubleshooting guide` (4b92966)
   - RAPL udev rule korrigiert
   - tmpfiles.d Konfiguration hinzugefügt
   - install.sh updated
   - TROUBLESHOOTING.md erstellt

2. `fix: downgrade charmbracelet/x/ansi to v0.10.0 for cellbuf compatibility` (2639bbb)
   - Dependency-Inkompatibilität behoben
   - Build funktioniert wieder

## Nächste Schritte

1. **Pull die Changes**:
   ```bash
   cd ~/aistack
   git pull
   ```

2. **Wähle eine der 3 Optionen** (siehe oben)

3. **Prüfe ob RAPL funktioniert**:
   ```bash
   sudo journalctl -u aistack-agent -f | grep rapl
   ```

4. **Für Idle Reset Problem**: Siehe `docs/TROUBLESHOOTING.md` Section "Idle State Constantly Resetting"

## Fragen?

Wenn das Problem weiterhin besteht:
```bash
# Erstelle Diagnostic Package
aistack diag --output /tmp/aistack-diag.zip

# Prüfe ob tmpfiles rule applied wurde
systemd-tmpfiles --create /etc/tmpfiles.d/aistack-rapl.conf --dry-run

# Prüfe Permissions
ls -la /sys/class/powercap/intel-rapl/intel-rapl:0/energy_uj
```
