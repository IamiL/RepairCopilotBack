import io
import mammoth
from fastapi import UploadFile, HTTPException
from typing import Dict, Any
from app.models.style_mapper import StyleMapper
from app.models.document_transformer import DocumentTransformer
from app.utils.logger import get_logger


class ConverterService:
    """
    Сервис для конвертации DOCX файлов в HTML
    """
    
    def __init__(self):
        self.logger = get_logger(__name__)
        self.style_mapper = StyleMapper()
        self.style_map = self.style_mapper.create_style_mapping()
        self.document_transformer = DocumentTransformer()
        
        self.logger.info("Инициализирован ConverterService")
        self.logger.debug("Создано маппингов стилей", 
                         style_mappings_count=len(self.style_map))
    
    async def convert_file(self, file: UploadFile) -> Dict[str, Any]:
        """
        Конвертирует DOCX файл в HTML
        
        Args:
            file: Загруженный DOCX файл
            
        Returns:
            Dict с результатом конвертации
        """
        self.logger.info("Начало конвертации файла", 
                         filename=file.filename,
                         content_type=file.content_type)
        
        if not file.filename:
            self.logger.error("Попытка конвертации без выбора файла")
            raise HTTPException(status_code=400, detail="Файл не выбран")
        
        if not file.filename.lower().endswith('.docx'):
            self.logger.error("Попытка конвертации неподдерживаемого формата",
                            filename=file.filename)
            raise HTTPException(
                status_code=400, 
                detail="Поддерживаются только DOCX файлы"
            )
        
        try:
            content = await file.read()
            file_size = len(content)
            file_like = io.BytesIO(content)
            
            self.logger.debug("Файл прочитан", 
                            filename=file.filename,
                            file_size=file_size)
            
            # Получаем опции трансформации
            transform_options = self.document_transformer.get_transform_options()
            
            self.logger.debug("Начало конвертации через Mammoth")
            # Конвертируем с использованием стилей и трансформации
            result = mammoth.convert_to_html(
                file_like, 
                style_map=self.style_map,
                **transform_options
            )
            html_content = result.value
            messages = result.messages
            
            warnings = [msg.message for msg in messages if msg.type == "warning"]
            errors = [msg.message for msg in messages if msg.type == "error"]
            
            self.logger.info("Mammoth конвертация завершена",
                           warnings_count=len(warnings),
                           errors_count=len(errors))
            
            # Анализ предупреждений о стилях
            self.logger.debug("Анализ стилевых предупреждений")
            style_analysis = self.style_mapper.analyze_style_warnings(warnings)
            warning_summary = self.style_mapper.get_warning_summary(style_analysis)
            
            complete_html = self._create_complete_html(html_content, file.filename)
            
            status = "success" if not errors else "success_with_errors"
            
            if warnings:
                self.logger.warning("Конвертация с предупреждениями",
                                  filename=file.filename,
                                  warnings_sample=warnings[:3])
            
            if errors:
                self.logger.error("Конвертация с ошибками",
                                filename=file.filename,
                                errors_sample=errors[:3])
            
            self.logger.info("Конвертация файла завершена успешно",
                           filename=file.filename,
                           status=status)
            
            return {
                "original_filename": file.filename,
                "file_size": file_size,
                "html_content": html_content,
                "complete_html": complete_html,
                "conversion_warnings": warnings,
                "conversion_errors": errors,
                "style_analysis": style_analysis,
                "warning_summary": warning_summary,
                "warnings_count": len(warnings),
                "errors_count": len(errors),
                "status": status
            }
            
        except Exception as e:
            self.logger.exception("Критическая ошибка при конвертации файла",
                                filename=file.filename,
                                error_type=type(e).__name__,
                                error_message=str(e))
            raise HTTPException(
                status_code=500, 
                detail=f"Ошибка при конвертации файла: {str(e)}"
            )
    
    def _create_complete_html(self, html_content: str, filename: str) -> str:
        """
        Создает полный HTML документ с необходимыми тегами
        
        Args:
            html_content: HTML контент от Mammoth
            filename: Имя исходного файла
            
        Returns:
            Полный HTML документ
        """
        self.logger.debug("Создание полного HTML документа", filename=filename)
        return f"""{html_content}"""