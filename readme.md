# Antifraud Processing — Browser Fingerprint Identification Service

Backend-сервис для идентификации браузерных профилей, кросс-браузерной идентификации устройств и выявления мультиаккаунтинга.

## Запуск

```bash
docker compose up --build
```

Сервис будет доступен на `http://localhost:8080`.

Миграции применяются автоматически при старте сервера.

## Переменные окружения


| Переменная        | Описание                                              | По умолчанию |
| ----------------- | ----------------------------------------------------- | ------------ |
| `DATABASE_URL`    | PostgreSQL connection string                          | —            |
| `HTTP_ADDR`       | Адрес HTTP-сервера                                    | `:8080`      |
| `MATCH_THRESHOLD` | Порог similarity score                                | `0.75`       |
| `MIGRATIONS_DIR`  | Директория с SQL-миграциями                           | `migrations` |
| `PRIVATE_KEY`     | RSA-2048 private key в формате PKCS#8, base64-encoded | —            |


### Генерация ключей

```bash
# RSA-2048 key pair
openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:2048 -outform DER -out private.der
openssl rsa -in private.der -inform DER -pubout -outform DER -out public.der

# Base64 для PRIVATE_KEY env
PRIVATE_KEY=$(base64 < private.der)

# Public key (DER, base64) передаётся JS-клиенту
PUBLIC_KEY=$(base64 < public.der)
```

Запуск с ключом:

```bash
PRIVATE_KEY=$(base64 < private.der) docker compose up --build
```

## API

### Health check

```bash
curl http://localhost:8080/health
```

### Resolve fingerprint

```
POST /api/v1/fingerprints/resolve
Content-Type: text/plain
```

Тело запроса — base64-строка с зашифрованными данными (см. [Протокол шифрования](#протокол-шифрования)).

Ответ:

```json
{
  "account_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567001",
  "browser_profile_id": "550e8400-e29b-41d4-a716-446655440000",
  "global_device_id": "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
  "match_type": "device_id",
  "score": 0.98,
  "device_linked_accounts_count": 2,
  "device_linked_accounts": ["a1b2c3d4-e5f6-7890-abcd-ef1234567001", "f9e8d7c6-b5a4-3210-fedc-ba9876543002"]
}
```

| Поле                           | Описание                                                                 |
|--------------------------------|--------------------------------------------------------------------------|
| `account_id`                   | ID аккаунта из запроса (UUID)                                            |
| `browser_profile_id`           | UUID профиля браузерного окружения                                       |
| `global_device_id`             | UUID кластера физического устройства                                     |
| `match_type`                   | Способ сопоставления: `device_id`, `hard_fp`, `soft_fp`, `candidate`, `new_profile` |
| `score`                        | Similarity score [0, 1]                                                  |
| `device_linked_accounts_count` | Количество уникальных аккаунтов на всех browser_profile одного устройства |
| `device_linked_accounts`       | Список account_id, привязанных к устройству                             |


---

## Протокол шифрования

Данные fingerprint шифруются на клиенте (JS) и передаются серверу как base64-строка в `text/plain`. Используется гибридная схема RSA-OAEP + AES-256-GCM:

```
┌─────────────────────────────────────────────────────────────┐
│  JS-клиент                                                  │
│                                                             │
│  1. Собрать fingerprint → JSON                              │
│  2. Сгенерировать случайный AES-256 ключ (32 байта)         │
│  3. Сгенерировать IV (12 байт)                              │
│  4. Зашифровать JSON с помощью AES-256-GCM(key, iv, json)   │
│  5. Зашифровать AES-ключ с помощью RSA-OAEP(SHA-256, pubkey)│
│  6. Склеить: encRsaKey(256) || iv(12) || encData(N)         │
│  7. Base64-encode → отправить POST text/plain               │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│  Go-сервер                                                  │
│                                                             │
│  1. Base64-decode тело запроса                              │
│  2. Разобрать: encRsaKey[0:256] | iv[256:268] | enc[268:]   │
│  3. RSA-OAEP decrypt(privkey, encRsaKey) → aesKey           │
│  4. AES-GCM decrypt(aesKey, iv, encData) → JSON             │
│  5. Unmarshal JSON → FingerprintInput                       │
│  6. Обработать fingerprint                                  │
└─────────────────────────────────────────────────────────────┘
```

Формат бинарных данных (до base64):


| Смещение   | Длина      | Содержимое                                  |
| ---------- | ---------- | ------------------------------------------- |
| `0..255`   | 256 байт   | AES-ключ, зашифрованный RSA-OAEP (SHA-256)  |
| `256..267` | 12 байт    | IV (nonce) для AES-GCM                      |
| `268..N`   | переменная | JSON fingerprint, зашифрованный AES-256-GCM |


---

## Сценарии проверки

> Кейсы показывают **расшифрованный JSON-payload** — содержимое, которое получается после декодирования base64 и расшифровки RSA-OAEP + AES-GCM. В production запросы отправляются только через JS-клиент, который выполняет шифрование автоматически.
>
> Для ручного тестирования без шифрования можно временно обойти слой шифрования (см. ниже).
>
> Все SHA-256 хеши и UUID реалистичны — 64 hex-символа / формат v4.

### Тестирование без шифрования

Для ручного тестирования через curl можно зашифровать JSON с помощью утилиты:

```bash
# Утилита шифрования (требует openssl и base64)
encrypt() {
  local json="$1" pubkey_der="$2"
  aes_key=$(openssl rand 32)
  iv=$(openssl rand 12)
  enc_aes_key=$(echo -n "$aes_key" | openssl pkeyutl -encrypt -pubin -inkey "$pubkey_der" -keyform DER -pkeyopt rsa_padding_mode:oaep -pkeyopt rsa_oaep_md:sha256)
  enc_data=$(echo -n "$json" | openssl enc -aes-256-gcm -K "$(echo -n "$aes_key" | xxd -p -c 64)" -iv "$(echo -n "$iv" | xxd -p -c 24)" -nosalt)
  (echo -n "$enc_aes_key"; echo -n "$iv"; echo -n "$enc_data") | base64
}
```

Либо использовать JS-клиент, который шифрует данные автоматически.

### Кейс 1: Первый вход — создание нового профиля

Пользователь впервые заходит на сайт с Windows-десктопа (Ryzen 7, RTX 3070 Ti, 2560×1440, 32 ГБ RAM) в Chrome.

Расшифрованный payload:

```json
{
  "account_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567001",
  "device_id": "d290f1ee-6c54-4b01-90e6-d701748f0851",
  "canvas_fp": "a3f2c8b1e9d74056af92c1d3e8b5f6a7104ec2d3f8a6b95072e1c4d5a9f3b8e7",
  "webgl_vendor": "Google Inc. (NVIDIA)",
  "webgl_renderer": "ANGLE (NVIDIA, NVIDIA GeForce RTX 3070 Ti, D3D11)",
  "audio_fp": "7c3e9a2f1b8d4056ce72a1f3d9e5b8c6203fa4e1d7b6c9850f3e2a1c8d7b6f54",
  "fonts_fp": "b5d2e8f1a3c7490682f1d3a5e9c7b6084d2f1a3e7c5b98064f2d1e3a7c5b9806",
  "webgl_params_hash": "e4c8a2f6d1b3950748e2c6a1f3d5b79082e4c8a6f2d1b395074e8c2a6f1d3b59",
  "webgl_extensions_hash": "19a3e7c2d5f8b406193e7a2c5d8f1b4061a3e7c2d5f8b40619e3a7c2d5f8b406",
  "math_fp": "2108f7d24dbe931609db9eb4a79c72d3265dcb816780d5f7e7a9c26c0aa86c0c",
  "media_codecs_hash": "4748d5437d8468316e497911586e0f76901d6a849625e7e355892b5e48fa6f7f",
  "user_agent_raw": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36",
  "user_agent_family": "chrome",
  "os_family": "windows",
  "cpu_cores": 16,
  "device_memory_gb": 32,
  "timezone_name": "Europe/Moscow",
  "language_code": "ru-RU",
  "languages": ["ru-RU", "ru", "en-US", "en"],
  "screen_width": 2560,
  "screen_height": 1440,
  "pixel_ratio": 1.0,
  "color_depth": 24,
  "max_touch_points": 0,
  "platform": "Win32",
  "avail_screen_width": 2560,
  "avail_screen_height": 1400,
  "do_not_track": null,
  "pdf_viewer_enabled": true,
  "color_gamut": "srgb",
  "hdr": true,
  "forced_colors": false,
  "prefers_color_scheme": "dark",
  "media_devices_count": {"audioinput": 2, "videoinput": 1, "audiooutput": 3},
  "timezone_offset": -180,
  "intl_locale": "ru-RU",
  "ip_addr": "178.154.200.41",
  "attrs": {"cookie_enabled": true}
}
```

Ожидаемый ответ:

```json
{
  "account_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567001",
  "browser_profile_id": "b3a1f7e2-4d5c-4e8a-9f1b-0c2d3e4f5a6b",
  "global_device_id": "d7e8f9a0-1b2c-4d3e-8f5a-6b7c8d9e0f1a",
  "match_type": "new_profile",
  "score": 0,
  "device_linked_accounts_count": 1,
  "device_linked_accounts": ["a1b2c3d4-e5f6-7890-abcd-ef1234567001"]
}
```

### Кейс 2: Мультиаккаунтинг — другой аккаунт, тот же браузер

Тот же ПК, тот же Chrome, но другой `account_id`. Все fingerprint-параметры идентичны, `device_id` совпадает → точный матч по `device_id`. Система фиксирует два разных аккаунта на одном профиле.

Расшифрованный payload:

```json
{
  "account_id": "f9e8d7c6-b5a4-3210-fedc-ba9876543002",
  "device_id": "d290f1ee-6c54-4b01-90e6-d701748f0851",
  "canvas_fp": "a3f2c8b1e9d74056af92c1d3e8b5f6a7104ec2d3f8a6b95072e1c4d5a9f3b8e7",
  "webgl_vendor": "Google Inc. (NVIDIA)",
  "webgl_renderer": "ANGLE (NVIDIA, NVIDIA GeForce RTX 3070 Ti, D3D11)",
  "audio_fp": "7c3e9a2f1b8d4056ce72a1f3d9e5b8c6203fa4e1d7b6c9850f3e2a1c8d7b6f54",
  "fonts_fp": "b5d2e8f1a3c7490682f1d3a5e9c7b6084d2f1a3e7c5b98064f2d1e3a7c5b9806",
  "webgl_params_hash": "e4c8a2f6d1b3950748e2c6a1f3d5b79082e4c8a6f2d1b395074e8c2a6f1d3b59",
  "webgl_extensions_hash": "19a3e7c2d5f8b406193e7a2c5d8f1b4061a3e7c2d5f8b40619e3a7c2d5f8b406",
  "math_fp": "2108f7d24dbe931609db9eb4a79c72d3265dcb816780d5f7e7a9c26c0aa86c0c",
  "media_codecs_hash": "4748d5437d8468316e497911586e0f76901d6a849625e7e355892b5e48fa6f7f",
  "user_agent_raw": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36",
  "user_agent_family": "chrome",
  "os_family": "windows",
  "cpu_cores": 16,
  "device_memory_gb": 32,
  "timezone_name": "Europe/Moscow",
  "language_code": "ru-RU",
  "languages": ["ru-RU", "ru", "en-US", "en"],
  "screen_width": 2560,
  "screen_height": 1440,
  "pixel_ratio": 1.0,
  "color_depth": 24,
  "max_touch_points": 0,
  "platform": "Win32",
  "avail_screen_width": 2560,
  "avail_screen_height": 1400,
  "do_not_track": null,
  "pdf_viewer_enabled": true,
  "color_gamut": "srgb",
  "hdr": true,
  "forced_colors": false,
  "prefers_color_scheme": "dark",
  "media_devices_count": {"audioinput": 2, "videoinput": 1, "audiooutput": 3},
  "timezone_offset": -180,
  "intl_locale": "ru-RU",
  "ip_addr": "178.154.200.41",
  "attrs": {"cookie_enabled": true}
}
```

Ожидаемый ответ:

```json
{
  "account_id": "f9e8d7c6-b5a4-3210-fedc-ba9876543002",
  "browser_profile_id": "b3a1f7e2-4d5c-4e8a-9f1b-0c2d3e4f5a6b",
  "global_device_id": "d7e8f9a0-1b2c-4d3e-8f5a-6b7c8d9e0f1a",
  "match_type": "device_id",
  "score": 0.99,
  "device_linked_accounts_count": 2,
  "device_linked_accounts": ["a1b2c3d4-e5f6-7890-abcd-ef1234567001", "f9e8d7c6-b5a4-3210-fedc-ba9876543002"]
}
```

### Кейс 3: Одно устройство, другой браузер (кросс-браузерная идентификация)

Тот же Windows-десктоп, но Firefox 138 вместо Chrome 146. Что меняется между браузерами:

- `device_id` — новый UUID (localStorage изолирован между браузерами)
- `canvas_fp` — другой хеш (Chrome рендерит через Skia/ANGLE, Firefox — через нативный OpenGL)
- `webgl_vendor` — Chrome: `"Google Inc. (NVIDIA)"`, Firefox: `"NVIDIA Corporation"`
- `webgl_renderer` — Chrome: `"ANGLE (..., D3D11)"`, Firefox: `"GeForce RTX 3070 Ti/PCIe/SSE2"`
- `webgl_params_hash`, `webgl_extensions_hash` — разные (ANGLE vs нативный GL)
- `media_codecs_hash` — разный (разные наборы поддерживаемых кодеков)
- `device_memory_gb` — `null` (Firefox не поддерживает `navigator.deviceMemory`)
- `intl_locale` — может отличаться (`"ru"` vs `"ru-RU"`)

Что остается одинаковым (hardware): `cpu_cores`, `screen_width/height`, `pixel_ratio`, `color_depth`, `max_touch_points`, `platform`, `os_family`, `timezone_name` → тот же `device_cluster`.

Расшифрованный payload:

```json
{
  "account_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567001",
  "device_id": "b7a1c3d4-e5f6-4890-abcd-ef1234567890",
  "canvas_fp": "6f1a3c5d7e9b2048af1c3e5d7b9a20486f1c3a5e7d9b20481f3a5c7e9d2b0486",
  "webgl_vendor": "NVIDIA Corporation",
  "webgl_renderer": "NVIDIA GeForce RTX 3070 Ti/PCIe/SSE2",
  "audio_fp": "7c3e9a2f1b8d4056ce72a1f3d9e5b8c6203fa4e1d7b6c9850f3e2a1c8d7b6f54",
  "fonts_fp": "b5d2e8f1a3c7490682f1d3a5e9c7b6084d2f1a3e7c5b98064f2d1e3a7c5b9806",
  "webgl_params_hash": "3a7c1e5d9b2f408637a1c5e9d3b7f2048a7c1e5d9b3f20486a7c1e5d9b2f4086",
  "webgl_extensions_hash": "d8b2f6a4c1e590738d2b6f4a1c5e908734d8b2f6a4c1e590783d2b6f4a1c5e90",
  "math_fp": "2108f7d24dbe931609db9eb4a79c72d3265dcb816780d5f7e7a9c26c0aa86c0c",
  "media_codecs_hash": "8b3f1a5c7d9e204861b3f5a7c9d1e20486b3f1a5c7d9e204868b3f1a5c7d9e20",
  "user_agent_raw": "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:138.0) Gecko/20100101 Firefox/138.0",
  "user_agent_family": "firefox",
  "os_family": "windows",
  "cpu_cores": 16,
  "device_memory_gb": null,
  "timezone_name": "Europe/Moscow",
  "language_code": "ru-RU",
  "languages": ["ru-RU", "ru", "en-US", "en"],
  "screen_width": 2560,
  "screen_height": 1440,
  "pixel_ratio": 1.0,
  "color_depth": 24,
  "max_touch_points": 0,
  "platform": "Win32",
  "avail_screen_width": 2560,
  "avail_screen_height": 1400,
  "do_not_track": "unspecified",
  "pdf_viewer_enabled": true,
  "color_gamut": "srgb",
  "hdr": true,
  "forced_colors": false,
  "prefers_color_scheme": "dark",
  "media_devices_count": {"audioinput": 2, "videoinput": 1, "audiooutput": 3},
  "timezone_offset": -180,
  "intl_locale": "ru",
  "ip_addr": "178.154.200.41",
  "attrs": {"cookie_enabled": true}
}
```

Ожидаемый ответ:

```json
{
  "account_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567001",
  "browser_profile_id": "e4c2a8d6-5f7b-4901-b3e2-1a9d8c7b6f5e",
  "global_device_id": "d7e8f9a0-1b2c-4d3e-8f5a-6b7c8d9e0f1a",
  "match_type": "new_profile",
  "score": 0,
  "device_linked_accounts_count": 2,
  "device_linked_accounts": ["a1b2c3d4-e5f6-7890-abcd-ef1234567001", "f9e8d7c6-b5a4-3210-fedc-ba9876543002"]
}
```

Что произошло:

- `browser_profile_id` — новый UUID (другой браузер → другие device_id, canvas, webgl)
- `global_device_id` — **тот же UUID** что в кейсах 1–2 (hardware_fp совпал: cpu_cores=16, screen 2560×1440, pixel_ratio=1, color_depth=24, max_touch_points=0, Win32, Europe/Moscow)
- `device_linked_accounts_count: 2` — на уровне устройства 2 уникальных аккаунта

### Кейс 4: Физически другое устройство (MacBook Pro M2 vs Windows-десктоп)

Другой компьютер — MacBook Pro 14" на Apple M2. Отличается всё: экран (1470×956 @ 2x vs 2560×1440 @ 1x), ОС (macos vs windows), GPU (Apple Metal vs NVIDIA D3D11), глубина цвета (30 vs 24), цветовая гамма (P3 vs sRGB), количество ядер (8 vs 16), RAM (8 vs 32 ГБ), media devices (1/1/1 vs 2/1/3).

Расшифрованный payload:

```json
{
  "account_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567001",
  "device_id": "c4e8f2a6-d1b3-4507-9e2c-6a1f3d5b7908",
  "canvas_fp": "82d72fc4203be63174efd7c76ceb10b0b863757a477e53b29070e7a4bbb37c9c",
  "webgl_vendor": "Google Inc. (Apple)",
  "webgl_renderer": "ANGLE (Apple, ANGLE Metal Renderer: Apple M2, Unspecified Version)",
  "audio_fp": "091904c9f55d694f467f40c6756ab939adc6a413b06b552d8897f7cd0f7393e7",
  "fonts_fp": "d0f04303bb4743e15fae2c55703dd8f3a98e8ee39337f66120e96ae2db3f7883",
  "webgl_params_hash": "8274de6b6d38cae67084463b6aa9b8eb82210cdc6cd10dbfcf3b7104b752fe4d",
  "webgl_extensions_hash": "45c0da2bb24f88a1ada4bde55fcf73de85a3d9fdee743e6817dbf6cffbca86f2",
  "math_fp": "2108f7d24dbe931609db9eb4a79c72d3265dcb816780d5f7e7a9c26c0aa86c0c",
  "media_codecs_hash": "f928c1d4e7a3b560f928c1d4e7a3b5601f928c1d4e7a3b560f9281cd4e7a3b56",
  "user_agent_raw": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36",
  "user_agent_family": "chrome",
  "os_family": "macos",
  "cpu_cores": 8,
  "device_memory_gb": 8,
  "timezone_name": "Europe/Moscow",
  "language_code": "en-GB",
  "languages": ["en-GB", "en-US", "en", "ru"],
  "screen_width": 1470,
  "screen_height": 956,
  "pixel_ratio": 2,
  "color_depth": 30,
  "max_touch_points": 0,
  "platform": "MacIntel",
  "avail_screen_width": 1470,
  "avail_screen_height": 919,
  "do_not_track": null,
  "pdf_viewer_enabled": true,
  "color_gamut": "p3",
  "hdr": true,
  "forced_colors": false,
  "prefers_color_scheme": "dark",
  "media_devices_count": {"audioinput": 1, "videoinput": 1, "audiooutput": 1},
  "timezone_offset": -180,
  "intl_locale": "en-GB",
  "ip_addr": "178.154.200.41",
  "attrs": {"cookie_enabled": true}
}
```

Ожидаемый ответ:

```json
{
  "account_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567001",
  "browser_profile_id": "a9f3b7c1-2e4d-4680-9a5c-8d1e0f3b7a2c",
  "global_device_id": "c5d6e7f8-9a0b-4c1d-2e3f-4a5b6c7d8e9f",
  "match_type": "new_profile",
  "score": 0,
  "device_linked_accounts_count": 1,
  "device_linked_accounts": ["a1b2c3d4-e5f6-7890-abcd-ef1234567001"]
}
```

### Кейс 5: Мультиаккаунтинг — другой аккаунт на втором устройстве

Второй аккаунт (который ранее использовался на Windows-десктопе в кейсе 2) теперь заходит с MacBook. `device_id` тот же что в кейсе 4 (тот же Chrome на том же MacBook) → матч по device_id. Система фиксирует мультиаккаунтинг: на этом browser_profile уже был другой аккаунт.

Расшифрованный payload:

```json
{
  "account_id": "f9e8d7c6-b5a4-3210-fedc-ba9876543002",
  "device_id": "c4e8f2a6-d1b3-4507-9e2c-6a1f3d5b7908",
  "canvas_fp": "82d72fc4203be63174efd7c76ceb10b0b863757a477e53b29070e7a4bbb37c9c",
  "webgl_vendor": "Google Inc. (Apple)",
  "webgl_renderer": "ANGLE (Apple, ANGLE Metal Renderer: Apple M2, Unspecified Version)",
  "audio_fp": "091904c9f55d694f467f40c6756ab939adc6a413b06b552d8897f7cd0f7393e7",
  "fonts_fp": "d0f04303bb4743e15fae2c55703dd8f3a98e8ee39337f66120e96ae2db3f7883",
  "webgl_params_hash": "8274de6b6d38cae67084463b6aa9b8eb82210cdc6cd10dbfcf3b7104b752fe4d",
  "webgl_extensions_hash": "45c0da2bb24f88a1ada4bde55fcf73de85a3d9fdee743e6817dbf6cffbca86f2",
  "math_fp": "2108f7d24dbe931609db9eb4a79c72d3265dcb816780d5f7e7a9c26c0aa86c0c",
  "media_codecs_hash": "f928c1d4e7a3b560f928c1d4e7a3b5601f928c1d4e7a3b560f9281cd4e7a3b56",
  "user_agent_raw": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36",
  "user_agent_family": "chrome",
  "os_family": "macos",
  "cpu_cores": 8,
  "device_memory_gb": 8,
  "timezone_name": "Europe/Moscow",
  "language_code": "en-GB",
  "languages": ["en-GB", "en-US", "en", "ru"],
  "screen_width": 1470,
  "screen_height": 956,
  "pixel_ratio": 2,
  "color_depth": 30,
  "max_touch_points": 0,
  "platform": "MacIntel",
  "avail_screen_width": 1470,
  "avail_screen_height": 919,
  "do_not_track": null,
  "pdf_viewer_enabled": true,
  "color_gamut": "p3",
  "hdr": true,
  "forced_colors": false,
  "prefers_color_scheme": "dark",
  "media_devices_count": {"audioinput": 1, "videoinput": 1, "audiooutput": 1},
  "timezone_offset": -180,
  "intl_locale": "en-GB",
  "ip_addr": "178.154.200.41",
  "attrs": {"cookie_enabled": true}
}
```

Ожидаемый ответ:

```json
{
  "account_id": "f9e8d7c6-b5a4-3210-fedc-ba9876543002",
  "browser_profile_id": "a9f3b7c1-2e4d-4680-9a5c-8d1e0f3b7a2c",
  "global_device_id": "c5d6e7f8-9a0b-4c1d-2e3f-4a5b6c7d8e9f",
  "match_type": "device_id",
  "score": 0.99,
  "device_linked_accounts_count": 2,
  "device_linked_accounts": ["a1b2c3d4-e5f6-7890-abcd-ef1234567001", "f9e8d7c6-b5a4-3210-fedc-ba9876543002"]
}
```

Второй аккаунт теперь привязан к двум устройствам (Windows-десктоп и MacBook) — мультиаккаунтинг виден на обоих.

---

## Анализируемые параметры

### Hard-признаки — поиск по индексу БД, уникальная идентификация в пределах браузера


| Параметр         | Источник (JS API)                                       | Описание                                                                                                                      | Зачем используется                                                                                                                                 |
| ---------------- | ------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------- |
| `device_id`      | `localStorage`                                          | UUID, генерируемый при первом визите и сохраняемый в localStorage. Уникален для пары (браузер + домен).                       | Самый надежный идентификатор в пределах одного браузера. Не работает кросс-браузерно, сбрасывается при очистке данных. Вес в similarity: **0.45**. |
| `canvas_fp`      | `HTMLCanvasElement.toDataURL()`                         | SHA-256 от результата рендера тестового рисунка на canvas. Разные GPU и драйверы дают разные субпиксельные результаты.        | Идентификация комбинации GPU + драйвер + браузер. Различается между браузерами на одном устройстве (разные рендер-движки).                         |
| `webgl_vendor`   | `WEBGL_debug_renderer_info` → `UNMASKED_VENDOR_WEBGL`   | Производитель GPU (например, `"Google Inc. (NVIDIA)"`). Chrome использует ANGLE и возвращает обёртку, Firefox — нативное имя. | Грубая идентификация GPU. Различается между Chrome (ANGLE) и Firefox (нативный OpenGL).                                                            |
| `webgl_renderer` | `WEBGL_debug_renderer_info` → `UNMASKED_RENDERER_WEBGL` | Модель GPU (например, `"ANGLE (NVIDIA, NVIDIA GeForce RTX 3070 Ti, D3D11)"`).                                                 | Точная модель GPU. Высокая энтропия, но зависит от браузера.                                                                                       |
| `hard_fp`        | Вычисляется на сервере                                  | SHA-256 от `device_id + canvas_fp + webgl_vendor + webgl_renderer + os_family + user_agent_family`.                           | Композитный хеш для быстрого поиска — если совпал, профиль почти наверняка тот же. Используется как индекс второго уровня после device_id.         |


### Medium-признаки — основное тело similarity score, высокая энтропия


| Параметр                | Источник (JS API)                                | Описание                                                                                                                                 | Энтропия | Стабильность  | Кросс-браузерность                                  | Вес      |
| ----------------------- | ------------------------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------- | -------- | ------------- | --------------------------------------------------- | -------- |
| `audio_fp`              | `OfflineAudioContext`                            | SHA-256 от результата рендера аудиосигнала. Зависит от аудио-стека ОС и аппаратного DAC.                                                 | ~35 бит  | Высокая       | Частичная (может отличаться между Chrome и Firefox) | **0.16** |
| `fonts_fp`              | Измерение ширины `<span>` с разными шрифтами     | SHA-256 от списка установленных шрифтов. Проверяется 37 шрифтов через измерение отклонения от базовых (monospace, sans-serif, serif).    | ~20 бит  | Высокая       | Высокая (шрифты — свойство ОС)                      | **0.14** |
| `webgl_params_hash`     | `WebGLRenderingContext.getParameter()`           | SHA-256 от значений `MAX_TEXTURE_SIZE`, `MAX_VERTEX_ATTRIBS`, `MAX_RENDERBUFFER_SIZE`, viewport dims и др. (9 параметров + 3 диапазона). | ~20 бит  | Высокая       | Низкая (ANGLE vs нативный GL дают разные значения)  | **0.08** |
| `webgl_extensions_hash` | `WebGLRenderingContext.getSupportedExtensions()` | SHA-256 от отсортированного списка поддерживаемых WebGL-расширений.                                                                      | ~15 бит  | Высокая       | Низкая (набор расширений зависит от браузера)       | **0.06** |
| `media_codecs_hash`     | `MediaSource.isTypeSupported()`                  | SHA-256 от битовой маски поддерживаемых кодеков (12 типов: avc1, hevc, vp8, vp9, av1, opus, vorbis, flac и т.д.).                        | ~8 бит   | Высокая       | Средняя                                             | **0.05** |
| `math_fp`               | `Math.tan()`, `Math.sin()`, `Math.cos()` и др.   | SHA-256 от результатов 7 математических функций с фиксированными аргументами. Разные JS-движки и FPU дают субъективно разные результаты. | ~5 бит   | Очень высокая | Средняя (V8 vs SpiderMonkey могут отличаться)       | **0.03** |
| `user_agent_family`     | Парсинг `navigator.userAgent`                    | Семейство браузера: `chrome`, `firefox`, `safari`, `edge`, `opera`, `yandex`, `other`.                                                   | ~3 бита  | Высокая       | Нет (по определению)                                | **0.04** |
| `os_family`             | Парсинг `navigator.userAgent`                    | Семейство ОС: `windows`, `macos`, `linux`, `android`, `ios`, `chromeos`, `other`.                                                        | ~3 бита  | Очень высокая | Да                                                  | **0.06** |


### Soft-признаки — стабильные числовые и строковые атрибуты


| Параметр           | Источник (JS API)                                  | Описание                                                                           | Энтропия | Стабильность                    | Кросс-браузерность    | Вес                      |
| ------------------ | -------------------------------------------------- | ---------------------------------------------------------------------------------- | -------- | ------------------------------- | --------------------- | ------------------------ |
| `soft_fp`          | Вычисляется на сервере                             | SHA-256 от всех medium + soft параметров (18 полей).                               | Высокая  | Средняя                         | Нет                   | **0.08**                 |
| `cpu_cores`        | `navigator.hardwareConcurrency`                    | Количество логических ядер CPU.                                                    | ~4 бита  | Очень высокая                   | Да                    | **0.03**                 |
| `device_memory_gb` | `navigator.deviceMemory`                           | Объём RAM в ГБ (округлённый). **Не поддерживается в Firefox** (возвращает `null`). | ~3 бита  | Очень высокая                   | Нет (только Chromium) | **0.02**                 |
| `timezone_name`    | `Intl.DateTimeFormat().resolvedOptions().timeZone` | IANA timezone (например, `"Europe/Moscow"`).                                       | ~9 бит   | Высокая (меняется при переезде) | Да                    | **0.02**                 |
| `language_code`    | `navigator.language`                               | Основной язык браузера (например, `"ru-RU"`).                                      | ~8 бит   | Высокая                         | Частичная             | **—** (входит в soft_fp) |
| `languages`        | `navigator.languages`                              | Полный список предпочитаемых языков.                                               | ~10 бит  | Средняя                         | Частичная             | **0.03**                 |
| `screen_width`     | `screen.width`                                     | Ширина экрана в CSS-пикселях.                                                      | ~7 бит   | Очень высокая                   | Да                    | **0.01**                 |
| `screen_height`    | `screen.height`                                    | Высота экрана в CSS-пикселях.                                                      | ~7 бит   | Очень высокая                   | Да                    | **0.01**                 |
| `pixel_ratio`      | `window.devicePixelRatio`                          | Соотношение физических и CSS-пикселей (1 для обычных дисплеев, 2 для Retina).      | ~3 бита  | Очень высокая                   | Да                    | **0.01**                 |
| `max_touch_points` | `navigator.maxTouchPoints`                         | Количество одновременных точек касания (0 для десктопов, 1–10 для тач-устройств).  | ~3 бита  | Очень высокая                   | Да                    | **0.02**                 |
| `color_depth`      | `screen.colorDepth`                                | Глубина цвета в битах (обычно 24 или 30).                                          | ~2 бита  | Очень высокая                   | Да                    | **0.01**                 |
| `platform`         | `navigator.platform`                               | Платформа ОС (например, `"Win32"`, `"MacIntel"`, `"Linux x86_64"`).                | ~3 бита  | Очень высокая                   | Да                    | **0.01**                 |


### Weak-признаки — дополнительные сигналы, хранятся в `variable_attrs` (jsonb)

Низкая энтропия по отдельности, но в совокупности добавляют ~11% к similarity score.


| Параметр                    | Источник (JS API)                                | Описание                                                               | Зачем                                                                                        | Вес       |
| --------------------------- | ------------------------------------------------ | ---------------------------------------------------------------------- | -------------------------------------------------------------------------------------------- | --------- |
| `avail_screen_width/height` | `screen.availWidth`, `screen.availHeight`        | Доступная область экрана (без панели задач).                           | Различает конфигурации рабочего стола на одинаковых мониторах.                               | **0.02**  |
| `media_devices_count`       | `navigator.mediaDevices.enumerateDevices()`      | Количество аудио-входов, видео-входов и аудио-выходов.                 | Уникальная комбинация подключённых устройств. Меняется при подключении наушников/веб-камеры. | **0.02**  |
| `color_gamut`               | `matchMedia("(color-gamut: ...)")`               | Цветовой охват дисплея: `srgb`, `p3`, `rec2020`.                       | Различает типы мониторов.                                                                    | **0.01**  |
| `timezone_offset`           | `new Date().getTimezoneOffset()`                 | Смещение UTC в минутах (например, -180 для MSK).                       | Дополняет timezone_name, помогает при подмене.                                               | **0.01**  |
| `intl_locale`               | `Intl.DateTimeFormat().resolvedOptions().locale` | Эффективная локаль форматирования (может отличаться от language_code). | Дополнительный языковой сигнал.                                                              | **0.01**  |
| `hdr`                       | `matchMedia("(dynamic-range: high)")`            | Поддержка HDR дисплеем.                                                | Различает HDR/SDR мониторы.                                                                  | **0.005** |
| `do_not_track`              | `navigator.doNotTrack`                           | Значение Do Not Track: `"1"`, `"0"`, `null`.                           | Пользовательская настройка, редко меняется.                                                  | **0.005** |
| `pdf_viewer_enabled`        | `navigator.pdfViewerEnabled`                     | Встроенный PDF-просмотрщик.                                            | Различает конфигурации браузера.                                                             | **0.005** |
| `forced_colors`             | `matchMedia("(forced-colors: active)")`          | Режим высокой контрастности (accessibility).                           | Редко включён, но стабилен.                                                                  | **0.005** |
| `prefers_color_scheme`      | `matchMedia("(prefers-color-scheme: dark)")`     | Тёмная/светлая тема ОС.                                                | Хранится, но не используется в score (слишком часто меняется).                               | **—**     |


### Hardware FP — кросс-браузерный идентификатор устройства


| Параметр            | Входит в `hardware_fp` | Почему                                                        |
| ------------------- | ---------------------- | ------------------------------------------------------------- |
| `cpu_cores`         | Да                     | `hardwareConcurrency` одинаков во всех браузерах              |
| `screen_width`      | Да                     | Физический экран не зависит от браузера                       |
| `screen_height`     | Да                     | —                                                             |
| `pixel_ratio`       | Да                     | Характеристика дисплея                                        |
| `max_touch_points`  | Да                     | Аппаратная характеристика тач-панели                          |
| `color_depth`       | Да                     | Характеристика дисплея                                        |
| `os_family`         | Да                     | ОС не меняется между браузерами                               |
| `platform`          | Да                     | `navigator.platform` стабилен                                 |
| `timezone_name`     | Да                     | Одинаков для всех браузеров на одном устройстве               |
| `webgl_vendor`      | **Нет**                | Chrome (ANGLE): `"Google Inc. (Apple)"` vs Firefox: `"Apple"` |
| `webgl_params_hash` | **Нет**                | Разные WebGL-бэкенды возвращают разные параметры              |
| `device_memory_gb`  | **Нет**                | Не поддерживается в Firefox (всегда `null`)                   |


---

## Архитектура

```
cmd/server/main.go          — точка входа
internal/config/             — загрузка конфигурации из env (DATABASE_URL, PRIVATE_KEY, ...)
internal/domain/             — модели, DTO, константы match_type
internal/fingerprint/        — вычисление hard_fp, soft_fp, hardware_fp, similarity score
internal/http/               — HTTP-хендлеры (chi router, CORS, base64 → decrypt → process)
internal/utils/              — RSA-OAEP + AES-256-GCM расшифровка, парсинг ключей
internal/repository/         — интерфейсы репозиториев (Profile, Event, Link, Cluster)
internal/service/            — бизнес-логика (ResolveOrCreateProfile, кластеризация)
internal/storage/postgres/   — реализация репозиториев на pgx/v5, транзакции, миграции
migrations/                  — SQL-миграции (up/down)
```

## Логика сопоставления

```
Запрос → ComputeHardFP / ComputeSoftFP / ComputeHardwareFP
         │
         ├─ 1. Поиск по device_id (индекс)
         │     Найден → match_type = "device_id", score = full similarity
         │
         ├─ 2. Поиск по hard_fp (индекс)
         │     Найден → match_type = "hard_fp", score = full similarity
         │
         ├─ 3. Поиск по soft_fp (индекс)
         │     Найден + soft score ≥ 0.75 → match_type = "soft_fp"
         │
         ├─ 4. Поиск кандидатов (os_family + cpu_cores + timezone + screen + touch)
         │     Лучший soft score ≥ 0.75 → match_type = "candidate"
         │
         └─ 5. Ничего не найдено → match_type = "new_profile"

После нахождения/создания browser_profile:
  → Найти/создать device_cluster по hardware_fp
  → Привязать профиль к кластеру
  → Upsert account_profile_link
  → Посчитать linked_accounts (на профиле и на кластере)
```

На этапах 3–4 similarity score считается только по medium/soft/weak признакам (без device_id и hard_fp), нормализованный к [0, 1].

Фильтр кандидатов (этап 4) включает `screen_width`, `screen_height` и `max_touch_points` для предотвращения ложных совпадений между физически разными устройствами (например, iPhone vs MacBook на одной ОС).

## Таблицы БД


| Таблица                 | Назначение                                                                            |
| ----------------------- | ------------------------------------------------------------------------------------- |
| `browser_profiles`      | Профиль браузерного окружения. Одна запись = один браузер на одном устройстве.        |
| `device_clusters`       | Кластер физического устройства. Группирует browser_profiles с одинаковым hardware_fp. |
| `fingerprint_events`    | Лог каждого входящего fingerprint-запроса.                                            |
| `account_profile_links` | Связь M:N между account_id и browser_profile_id.                                      |


