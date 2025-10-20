import logging
import sys
from typing import Dict, Any
from datetime import datetime
import json


class JSONFormatter(logging.Formatter):
    """
    Кастомный форматтер для структурированного JSON логирования
    """
    
    def format(self, record: logging.LogRecord) -> str:
        log_data = {
            "timestamp": datetime.utcnow().isoformat() + "Z",
            "level": record.levelname,
            "logger": record.name,
            "message": record.getMessage(),
            "module": record.module,
            "function": record.funcName,
            "line": record.lineno
        }
        
        if hasattr(record, 'extra_data'):
            log_data.update(record.extra_data)
            
        if record.exc_info:
            log_data["exception"] = self.formatException(record.exc_info)
            
        return json.dumps(log_data, ensure_ascii=False, separators=(',', ':'))


class LoggerConfig:
    """
    Конфигурация системы логирования
    """
    
    @staticmethod
    def setup_logging(
        level: str = "INFO",
        format_type: str = "json",
        log_to_file: bool = False,
        log_file_path: str = "app.log"
    ) -> None:
        """
        Настройка системы логирования
        
        Args:
            level: Уровень логирования (DEBUG, INFO, WARNING, ERROR, CRITICAL)
            format_type: Тип форматирования ('json' или 'standard')
            log_to_file: Логировать в файл
            log_file_path: Путь к файлу логов
        """
        log_level = getattr(logging, level.upper(), logging.INFO)
        
        # Очищаем существующие обработчики
        root_logger = logging.getLogger()
        for handler in root_logger.handlers[:]:
            root_logger.removeHandler(handler)
        
        # Настраиваем форматтер
        if format_type == "json":
            formatter = JSONFormatter()
        else:
            formatter = logging.Formatter(
                '%(asctime)s - %(name)s - %(levelname)s - %(message)s'
            )
        
        # Консольный обработчик
        console_handler = logging.StreamHandler(sys.stdout)
        console_handler.setFormatter(formatter)
        console_handler.setLevel(log_level)
        
        # Настройка корневого логгера
        root_logger.setLevel(log_level)
        root_logger.addHandler(console_handler)
        
        # Файловый обработчик (если необходимо)
        if log_to_file:
            file_handler = logging.FileHandler(log_file_path, encoding='utf-8')
            file_handler.setFormatter(formatter)
            file_handler.setLevel(log_level)
            root_logger.addHandler(file_handler)
        
        # Отключаем избыточное логирование библиотек
        logging.getLogger("uvicorn.access").setLevel(logging.WARNING)
        logging.getLogger("uvicorn.error").setLevel(logging.INFO)


class StructuredLogger:
    """
    Обертка для структурированного логирования с дополнительными данными
    """
    
    def __init__(self, name: str):
        self.logger = logging.getLogger(name)
    
    def _log_with_extra(self, level: int, message: str, extra_data: Dict[str, Any] = None):
        """
        Логирование с дополнительными структурированными данными
        """
        if extra_data:
            extra = {"extra_data": extra_data}
            self.logger.log(level, message, extra=extra)
        else:
            self.logger.log(level, message)
    
    def debug(self, message: str, **kwargs):
        """Debug уровень логирования"""
        self._log_with_extra(logging.DEBUG, message, kwargs)
    
    def info(self, message: str, **kwargs):
        """Info уровень логирования"""
        self._log_with_extra(logging.INFO, message, kwargs)
    
    def warning(self, message: str, **kwargs):
        """Warning уровень логирования"""
        self._log_with_extra(logging.WARNING, message, kwargs)
    
    def error(self, message: str, **kwargs):
        """Error уровень логирования"""
        self._log_with_extra(logging.ERROR, message, kwargs)
    
    def critical(self, message: str, **kwargs):
        """Critical уровень логирования"""
        self._log_with_extra(logging.CRITICAL, message, kwargs)
    
    def exception(self, message: str, **kwargs):
        """Логирование исключения с трассировкой стека"""
        if kwargs:
            extra = {"extra_data": kwargs}
            self.logger.exception(message, extra=extra)
        else:
            self.logger.exception(message)


def get_logger(name: str) -> StructuredLogger:
    """
    Получить структурированный логгер
    
    Args:
        name: Имя логгера (обычно __name__)
        
    Returns:
        Экземпляр StructuredLogger
    """
    return StructuredLogger(name)