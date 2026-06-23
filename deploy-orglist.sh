#!/bin/bash
# Выкатить orglist.txt из этого репо на узлы splitroute (panda + se1) и
# перегенерировать наборы (panda→wgse1, se1→viavvp ipset).
#
# Зачем так, а не git pull/curl на самих узлах: узлы на RU-egress, доступ к github
# (raw/api/git) по тому же пути через vvp нестабилен — github рвёт тело ответа.
# Поэтому источник-версионник = этот репо (github+локально), а на узлы orglist
# ВЫКАТЫВАЕТСЯ с управляющего хоста (WSL), который достаёт узлы стабильно.
#
# Запускать с WSL после правки orglist.txt:  ./deploy-orglist.sh
set -e
cd "$(dirname "$0")"
[ -s orglist.txt ] || { echo "нет orglist.txt"; exit 1; }
echo "orglist ($(grep -c . orglist.txt) орг) → panda + se1"

echo "=== panda (нажми юбик на sudo) ==="
ssh -A kscht@roslavl.404-net.ru \
  'sudo -E bash -c "cat > /opt/splitroute/orglist.txt; systemctl start splitroute-refresh.service; grep -E \"orglist:|applied:\" /var/log/splitroute-refresh.log | tail -2"' \
  < orglist.txt

echo "=== se1 ==="
ssh se1 \
  'cat > /opt/splitroute/orglist.txt; systemctl start splitroute-viavvp-refresh.service; grep -E "orglist:|done:" /var/log/splitroute-viavvp-refresh.log | tail -2' \
  < orglist.txt

echo "=== готово. Не забудь зафиксировать историю: git add orglist.txt && git commit && git push ==="
