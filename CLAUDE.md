# CLAUDE.md

## Что делает проект

Генератор статических маршрутов для роутеров Keenetic. Скачивает базу ipinfo.io (`country_asn.json.gz`), находит CIDRs нужных организаций и оптимизирует их — не захватывая при этом российские IP-адреса. На выходе: команды `ip route` для Keenetic и чистый список CIDRs для других систем.

## Структура

```
main.go              — CLI-точка входа, загрузка config.toml
fetch.go             — скачать базу, распарсить JSONL, сохранить CIDRs
optimize.go          — оптимизировать CIDRs без захвата RU-сетей
routes.go            — сгенерировать маршруты Keenetic + cidr.txt
config.toml          — конфиг (gitignored, содержит токен)
config.example.toml  — шаблон конфига
orglist.txt          — список доменов организаций (as_domain из ipinfo)
```

Промежуточные файлы в `.cache/` (gitignored):
- `country_asn.json.gz` — база ipinfo (TTL 24 ч)
- `ru_networks.txt` — RU-сети, нужны оптимизатору

## Команды

```bash
go build -o splitroute .
./splitroute all       # fetch → optimize → routes
./splitroute fetch     # скачать базу, распарсить
./splitroute optimize  # оптимизировать CIDRs
./splitroute routes    # сгенерировать маршруты
```

## Архитектура `optimize.go`

Ключевые структуры данных:
- `[]netip.Prefix` — сети, отсортированные по первому адресу
- RU-сети хранятся отсортированными → `sort.Search` даёт O(log n) при проверке

Алгоритм `optimizeNets`:
1. `dedupe` — убрать подсети и дубликаты
2. Цикл по `targetBits` от 11 до 22:
   - `expandAll` — расширить каждую сеть до `targetBits`, если не захватывает RU/private
   - `collapse` — итеративно объединять соседние сети-сиблинги до стабилизации

Проверка захвата RU (`newRU` / `newRUMerge`): binary search по первому адресу кандидатов, затем `subnetOf` для точной проверки.

## Форматы файлов

`networks.txt` / `optimized_networks.txt` / `cidr.txt` — один CIDR на строку:
```
1.0.0.0/10
1.96.0.0/11
```

`routes.txt` — команды Keenetic:
```
ip route 1.0.0.0 255.192.0.0 192.168.99.1 OpenConnect0 auto reject
```

## Конфиг (`config.toml`)

```toml
[api]
token = "..."

[files]
org_list         = "orglist.txt"
networks_output  = "networks.txt"
optimized_output = "optimized_networks.txt"
routes_output    = "routes.txt"
cidr_output      = "cidr.txt"      # пусто — не записывать

[routing]
gateway_ip   = "192.168.99.1"
gateway_name = "OpenConnect0"
route_type   = "auto reject"
```

## Зависимости

Единственная внешняя: `github.com/BurntSushi/toml v1.4.0`. Всё остальное — stdlib (`net/netip`, `compress/gzip`, `encoding/json`, `bufio`, `sort`).
