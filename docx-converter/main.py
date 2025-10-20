import uvicorn
from fastapi import FastAPI
from app.controllers.converter_controller import router as converter_router
from app.utils.logger import LoggerConfig, get_logger


def create_app() -> FastAPI:
    # Настройка логирования
    LoggerConfig.setup_logging(
        level="INFO",
        format_type="json",
        log_to_file=True,
        log_file_path="docx_converter.log"
    )
    
    logger = get_logger(__name__)
    logger.info("Инициализация приложения Document Converter API", version="1.0.0")
    
    app = FastAPI(
        title="Document Converter API",
        description="API для конвертации документов",
        version="1.0.0"
    )
    
    app.include_router(converter_router, prefix="/api/v1")
    logger.info("Маршруты зарегистрированы", prefix="/api/v1")
    
    return app


if __name__ == "__main__":
    logger = get_logger(__name__)
    logger.info("Запуск сервера", host="0.0.0.0", port=8000)
    
    app = create_app()
    uvicorn.run(app, host="0.0.0.0", port=8000)