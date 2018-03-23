# Changelog

## [v0.10] - 23-03-2017

### Added

#### stop-sync
- Функция синхронного ожидания закрытия потока логгера

## [v0.4.0] - 11-09-2017

### Added

#### WNE-460-split-long-messages
- Разделение длинных сообщений на несколько

#### add-writers
- Добавление возможности задавать дополнительный вывод лога (например, в файлы)

## [v0.3.0] - 11-08-2017

### Added

#### 0.3.0
- Support for io.Writer and log.Logger

### Changed

#### 0.3.0
- Critical -> Error; ErrorLevel -> LogLevel
- Environment variable renamed from LogLevel to LOGLEVEL
- Logger.Level is now public, so it can be changed in runtime

## [v0.0.2] - 16-06-2017

### Added

#### v0.0.2
- Функции Debug/Info/Warning/Critical для вывода информации в json-формате
- Singleton- и Object-based функции
- Конфигурирование вывода при помощи переменной окружения LOGLEVEL
- Логгирование в отдельном потоке
