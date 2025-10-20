from typing import Dict, Any
from app.services.mapping_service import MappingService


class ConverterService:
    """
    Сервис для конвертации HTML в Markdown с маппингом
    """
    
    def __init__(self):
        self.mapping_service = MappingService()
    
    async def convert_html_text(self, html_text: str) -> Dict[str, Any]:
        """
        Конвертирует HTML текст в Markdown с созданием маппинга
        
        Args:
            html_text: HTML текст для конвертации
            
        Returns:
            Dict с результатом конвертации и маппингом
        """
        result = self.mapping_service.convert_with_mapping(html_text)
        return result
    
    async def find_html_by_line(self, line_number: int) -> Dict[str, Any]:
        """
        Находит HTML блок по номеру строки Markdown
        
        Args:
            line_number: Номер строки в Markdown документе
            
        Returns:
            Dict с информацией о HTML блоке
        """
        return self.mapping_service.find_html_by_line(line_number)