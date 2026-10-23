<img src="assets/icon.png" alt="Иконка Battle City" width="96" align="right">

# Battle City

[English](README.md)

Защищайте базу, уничтожайте вражеские танки и пройдите 35 классических уровней.
Играйте в одиночку или вдвоём на одном компьютере. Настольная версия Battle City
для Windows, Linux и macOS сохраняет исходные пиксельную графику, карты и звуковые данные,
добавляя управление с клавиатуры и геймпадов.

## Скриншоты

| Главное меню | Выбор уровня |
|---|---|
| ![Главное меню](screenshots/menu.png) | ![Выбор уровня](screenshots/stage-selection.png) |
| Один игрок | Совместная игра |
| ![Игра в одиночку](screenshots/gameplay.png) | ![Игра вдвоём](screenshots/cooperative.png) |

## Возможности

- 35 уровней, режимы для одного и двух игроков, исходный конструктор карт.
- Оригинальные графика и звуковые данные; увеличение изображения без сглаживания.
- Клавиатура и стандартные геймпады для обоих игроков.
- Английская подсказка выбора STAGE, выполненная основным игровым шрифтом.
- Полноэкранный режим, выбор NTSC/PAL и режим квадратных пикселей.
- Сохранение состояния, автоматическое сохранение рекорда и возврат в меню по Esc.
- Сборка для macOS Intel (amd64) и Apple Silicon (arm64), упаковка в `.app` с иконкой.
- Иконка с танком для окна приложения, Windows EXE и приложения macOS.

## Запуск

Распакуйте архив и запустите:

- **Windows x64:** `bin/BattleCityMD-windows-amd64.exe`.
- **Linux x64:** `./bin/BattleCityMD-linux-amd64` в графической сессии.
  Нужны X11/XWayland, OpenGL и библиотеки PulseAudio или ALSA.
- **macOS Intel:** `bin/BattleCityMD-macos-amd64.app`.
- **macOS Apple Silicon:** `bin/BattleCityMD-macos-arm64.app`.
  Откройте соответствующее приложение в Finder. Требуется macOS 12 или новее.
  Готовые пакеты не проходили нотариализацию Apple.

Для готовых сборок не требуются Go, отдельный эмулятор или подключение к интернету.
В меню Enter выбирает пункт. Для совместной игры выберите **2 PLAYERS**.

### Управление

| Действие | Игрок 1 | Игрок 2 |
|---|---|---|
| Движение | Стрелки | W A S D |
| Огонь / кнопки A, B, C | Z / X / C | G / H / J |
| Дополнительная кнопка огня / B | Пробел | — |
| Start / пауза в игре | Enter | Tab |

На экране **STAGE**: **Enter** — подтвердить, **Пробел** или **X** — следующий
уровень, **C** — предыдущий уровень.

Геймпады: крестовина или левый стик, лицевые кнопки West/South/East и Start.

| Клавиша | Действие |
|---|---|
| Esc | Выход из программы в главном меню; возврат в главное меню с остальных экранов |
| F1 | Подсказка управления |
| F5 / F9 | Сохранить / загрузить состояние |
| F8 | Перезапустить консоль |
| F11 | Полноэкранный режим |
| M | Выключить / включить звук |
| F10 | Выход из программы |

### Параметры

```sh
BattleCityMD-windows-amd64.exe -fullscreen
BattleCityMD-windows-amd64.exe -pal
BattleCityMD-windows-amd64.exe -aspect pixel -scale 4
```

По умолчанию: NTSC, 60 обновлений в секунду, область изображения 4:3 и увеличение
без сглаживания. `-pal` выбирает 50 обновлений в секунду; `-aspect pixel` включает
квадратные пиксели с целочисленным увеличением; `-scale 1..6` задаёт начальный
размер окна; `-mute` запускает игру без звука.

Рекорд и сохранения находятся в папке `BattleCityMD-Go` пользовательского каталога
настроек: `%APPDATA%` в Windows, `$XDG_CONFIG_HOME` или `~/.config` в Linux,
`~/Library/Application Support` в macOS.
Для каждого региона используется один слот. Сохранения версии 0.1.1 совместимы.

## Сборка из исходников

Требуется **Go 1.25.7+**. Все команды выполняются из корня проекта с `main.go`.
Зависимости загружаются через Go modules; папки `vendor` нет.
Для настольных платформ используется Ebitengine 2.10.5 с `CGO_ENABLED=0`.
Для компиляции не нужны C-компилятор, Xcode и заголовочные файлы системных библиотек.

Windows PowerShell:

```powershell
go mod download
$env:CGO_ENABLED="0"
go build -trimpath -ldflags="-s -w -H=windowsgui" -o BattleCityMD.exe .
```

Linux:

```sh
go mod download
CGO_ENABLED=0 go build -trimpath -o BattleCityMD .
./BattleCityMD
```

Приложения macOS из **Windows, Linux или macOS**:

```sh
go run build_macos.go
```

Команда собирает обе архитектуры: `bin/BattleCityMD-macos-amd64.app` и
`bin/BattleCityMD-macos-arm64.app`. Включает `Info.plist` и иконку ICNS.
Для сборки одной архитектуры:

```sh
go run build_macos.go -arch amd64
go run build_macos.go -arch arm64
```

Запускайте сборщик обычным Go для текущего компьютера. Целевую платформу он
задаёт автоматически для дочерних процессов. На macOS при наличии `codesign`
добавляется локальная ad-hoc подпись; учётная запись разработчика Apple не нужна.

Отдельные исполняемые файлы macOS без `.app` (Bash/Zsh):

```sh
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -trimpath -o bin/BattleCityMD-macos-amd64 .
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -o bin/BattleCityMD-macos-arm64 .
```

Файл `icon_windows_amd64.syso` уже содержит ресурс иконки для Windows.
Из сборки macOS он исключается автоматически. Изображения иконки:
`assets/icon.png`, `assets/icon.ico` и `assets/icon.icns`.

## Реализация и лицензия

Приложение исполняет программу картриджа через Go-ядро Mega Drive. Исходные
игровые ресурсы сохранены. Это не полное переписывание всех игровых правил на Go;
тактовая идентичность аппаратной консоли не заявляется.

Приложение и адаптированное ядро: **GPL-3.0**, см. [LICENSE](LICENSE).
Исходные игровые ресурсы сохраняют принадлежность своим правообладателям.
Сторонние компоненты перечислены в [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md).
