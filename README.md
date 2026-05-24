# splitroute

Генератор статических маршрутов для роутеров Keenetic. Скачивает базу IP-адресов ipinfo.io, находит сети указанных организаций (Telegram, Google и т.д.) и генерирует оптимизированные маршруты — так, чтобы трафик к этим организациям шёл через выбранный шлюз. Соседние блоки адресов объединяются и расширяются, чтобы минимизировать итоговое количество маршрутов, при этом российские IP-адреса намеренно исключаются из агрегации и в маршруты не попадают.

## Требования

- Linux (amd64 / arm64 / armv7)
- Токен [ipinfo.io](https://ipinfo.io) (бесплатный тариф подходит)

## Установка

Go устанавливать не нужно — скрипт сделает это автоматически.

```bash
git clone https://github.com/kscht/splitroute.git
cd splitroute
bash install.sh   # установит Go при необходимости, соберёт и скопирует бинарник в /usr/local/bin
```

## Настройка

```bash
cp config.example.toml config.toml
```

Отредактируйте `config.toml`:

```toml
[api]
token = "ваш_токен_ipinfo"

[files]
org_list         = "orglist.txt"       # список организаций
networks_output  = "networks.txt"      # найденные CIDRs
optimized_output = "optimized_networks.txt"
routes_output    = "routes.txt"        # команды для Keenetic
cidr_output      = "cidr.txt"          # чистые CIDRs для других систем

[routing]
gateway_ip   = "192.168.99.1"          # IP шлюза
gateway_name = "OpenConnect0"          # имя интерфейса
route_type   = "auto reject"
```

Добавьте нужные организации в `orglist.txt` (по одному домену на строку):

```
telegram.org
google.com
cloudflare.com
```

## Использование

```bash
./splitroute all       # полный цикл: fetch → optimize → routes
./splitroute fetch     # скачать базу, извлечь CIDRs организаций
./splitroute optimize  # оптимизировать (объединить и расширить блоки)
./splitroute routes    # сгенерировать маршруты
./splitroute validate      # проверить корректность оптимизации
./splitroute lookup <ip>   # проверить, попадает ли IP в маршруты
```

### Результат

| Файл | Содержимое |
|------|-----------|
| `networks.txt` | все найденные CIDRs организаций |
| `optimized_networks.txt` | оптимизированные CIDRs |
| `routes.txt` | команды `ip route` для Keenetic CLI |
| `cidr.txt` | те же CIDRs, один на строку — для iptables, nftables, OpenWRT и др. |

Пример строки в `routes.txt`:
```
ip route 1.0.0.0 255.192.0.0 192.168.99.1 OpenConnect0 auto reject
```

### Проверка IP

```
$ splitroute lookup 8.8.8.8
HIT  8.8.8.8  →  8.0.0.0/10

$ splitroute lookup 1.2.3.4
MISS 1.2.3.4
```

Команда читает локальный `optimized_networks.txt` и отвечает, через какой маршрут пойдёт адрес, или `MISS` если он не охвачен.

## Логика оптимизации

Оптимизатор объединяет и расширяет CIDR-блоки организаций с двумя ограничениями:
- не захватывать приватные диапазоны (RFC 1918 и др.)
- не захватывать российские IP-адреса (страна `RU` по базе ipinfo)

Типичный результат: ~196 000 исходных блоков → ~2 100 оптимизированных (сжатие ~92x).

## База данных

При первом запуске скачивается `country_asn.json.gz` (~30 МБ) с ipinfo.io и кешируется в `.cache/` на 24 часа.

## Лицензия

MIT
