#!/bin/bash
# Complete Fix Script for aistack suspend issues

set -e

echo "=== Fix 1: Docker Group ==="
sudo usermod -aG docker $USER
echo "✓ Added $USER to docker group (logout/login required)"

echo ""
echo "=== Fix 2: RAPL Permissions ==="
sudo systemd-tmpfiles --create /etc/tmpfiles.d/aistack-rapl.conf
echo "✓ RAPL permissions applied"

echo ""
echo "=== Fix 3: Rebuild Binary ==="
cd ~/aistack
make build
sudo cp ./dist/aistack /usr/local/bin/aistack
echo "✓ Binary rebuilt and installed"

echo ""
echo "=== Fix 4: Check Timer Configuration ==="
sudo cat /etc/systemd/system/aistack-idle.timer

echo ""
echo "=== Fix 5: Enable and Start Timer ==="
sudo systemctl daemon-reload
sudo systemctl enable aistack-idle.timer
sudo systemctl start aistack-idle.timer
sudo systemctl status aistack-idle.timer --no-pager
echo "✓ Timer enabled and started"

echo ""
echo "=== Fix 6: Check Service Configuration ==="
sudo cat /etc/systemd/system/aistack-idle.service | grep ExecStart

echo ""
echo "=== All fixes applied! ==="
echo ""
echo "Next steps:"
echo "1. Logout and login again (for docker group)"
echo "2. Check timer status: sudo systemctl status aistack-idle.timer"
echo "3. Check agent logs: sudo journalctl -u aistack-agent -f"
echo "4. Test idle-check: sudo aistack idle-check --ignore-inhibitors"
