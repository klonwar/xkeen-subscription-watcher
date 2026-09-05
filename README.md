# xkeen-subscription-watcher

[![Test](https://github.com/klonwar/xkeen-subscription-watcher/actions/workflows/test.yml/badge.svg)](https://github.com/klonwar/xkeen-subscription-watcher/actions/workflows/test.yml)
[![codecov](https://codecov.io/gh/klonwar/xkeen-subscription-watcher/graph/badge.svg)](https://codecov.io/gh/klonwar/xkeen-subscription-watcher)

## Использование

```bash
xkeen-subscription-watcher <tag>=<url>
```

Можно передать несколько пар `<tag>=<url>`

Скрипт получает из переданных подписок прокси, генерирует конфиги `04_outbounds.<tag>.json`.

Если конфиги изменились, выполняется `xkeen -restart`.

### Поддерживаемые протоколы

- `vless` (reality / tls / none; транспорты `tcp`, `raw`, `ws`, `grpc`, `xhttp`)
- `vmess` (формат `vmess://base64(JSON)`)
- `trojan`
- `shadowsocks` (`ss`)
- `hysteria2`

Для `vmess` и `trojan` поддерживаются транспорты `tcp`, `raw`, `ws`, `grpc`.
Узлы с транспортами, которые удалены из актуального Xray (`h2`/`http`, `quic`,
`kcp` и т.п.), пропускаются с предупреждением в логе — чтобы один такой узел не
ломал весь конфиг.

### Флаги

- `--output-dir <path>` — каталог для конфигов (по умолчанию `/opt/etc/xray/configs`)
- `--no-restart` — не перезапускать XKeen после обновления
- `--single-proxy` — брать только первый прокси из подписки
- `--reality-fingerprint <fp>` — переопределить fingerprint для Reality
- `--dialer-proxies=<proxy1>,<proxy2>` — dialer proxies через запятую
- `--include-name <pattern>` — оставлять узлы с указанной подстрокой в имени
- `--include-name-glob <pattern>` — оставлять узлы по glob-маске имени
- `--exclude-name <pattern>` — исключать узлы с указанной подстрокой в имени
- `--exclude-name-glob <pattern>` — исключать узлы по glob-маске имени
- `--include-protocol <protocol>` / `--exclude-protocol <protocol>` — фильтр по протоколу
- `--include-transport <transport>` / `--exclude-transport <transport>` — фильтр по транспорту
- `--limit <N>` — ограничить число оставшихся узлов

Новые флаги можно повторять. Значение без `tag=` применяется ко всем подпискам,
а форма `tag=value` — только к указанному тегу. Например, чтобы настроить разные
ограничения в одном запуске:

```shell
xkeen-subscription-watcher \
  home=https://example.com/home \
  work=https://example.com/work \
  --include-name home=Germany \
  --include-name-glob work=FI-* \
  --include-protocol home=vless \
  --include-transport work=grpc \
  --exclude-name-glob home='*test*' \
  --limit home=3 \
  --limit work=5
```

`--include-name` и `--exclude-name` ищут подстроку без учета регистра.
Для glob-масок используйте отдельные флаги и заключайте значения с `*`, `?` или
скобками в кавычки, чтобы shell не раскрыл их до запуска программы. Доступны
протоколы `vless`, `vmess`, `trojan`, `ss`, `hysteria2` и транспорты `tcp`,
`raw`, `ws`, `grpc`, `xhttp`, `hysteria`.

## Установка

```shell
curl -sSf https://raw.githubusercontent.com/klonwar/xkeen-subscription-watcher/master/install.sh | sh
```

Или вручную — скачать бинарник для своей архитектуры из [Releases](https://github.com/klonwar/xkeen-subscription-watcher/releases/latest):

```shell
curl -sSLo /opt/sbin/xkeen-subscription-watcher <url-бинарника>
chmod +x /opt/sbin/xkeen-subscription-watcher
```

### Crontab

```shell
crontab -e
```

Добавляем что-то вроде:

```crontab
0 * * * * /opt/sbin/xkeen-subscription-watcher <tag>=<url>
```

### Настройка Xray

Убираем из `04_outbounds.json` прокси, которые будут теперь генерироваться из подписок,
иначе теги будут конфликтовать, оставляем например такое:

```json
{
  "outbounds": [
    {
      "tag": "direct",
      "protocol": "freedom"
    },
    {
      "tag": "block",
      "protocol": "blackhole",
      "response": {
        "type": "HTTP"
      }
    }
  ]
}
```

В `05_routing.json` добавляем конфигурацию `balancers` и `burstObservatory` для автоматического выбора лучшего прокси,
далее в `rules` используем `balancerTag` вместо `outboundTag`, например:

```json
{
  "routing": {
    "domainStrategy": "AsIs",
    "balancers": [
      {
        "tag": "proxy",
        "selector": ["<tag>"],
        "strategy": {
          "type": "leastPing"
        }
      }
    ],
    "rules": [
      {
        "inboundTag": ["socks", "http"],
        "balancerTag": "proxy"
      }
    ]
  },
  "burstObservatory": {
    "subjectSelector": ["<tag>"],
    "pingConfig": {}
  }
}
```

Разово запускаем команду из crontab, проверяем, что всё работает.
