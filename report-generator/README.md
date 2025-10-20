# Генератор отчёта об ошибках ТЗ (.docx)

HTTP-сервис на FastAPI, который принимает JSON с секциями и ошибками и возвращает Word-документ с отчётом.  
Поля **llm_id** и **risks** для каждой ошибки добавляются в **комментарии (Review)**, привязанные к фразе *«Что некорректно»*.

## Запуск

### В Docker
```bash
docker compose up --build
# или
docker build -t tz-report .
docker run --rm -p 8000:8000 tz-report
