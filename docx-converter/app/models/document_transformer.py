import mammoth
import mammoth.transforms
from typing import Any
from app.utils.logger import get_logger


class DocumentTransformer:
    """
    Класс для трансформации документов перед конвертацией в HTML
    """
    
    def __init__(self):
        self.logger = get_logger(__name__)
        self.undefined_style_ids = [
            'Style2', 'Style4', 'Style18', '20', '1', 'a9', 
            'ab', '24', '11', '30'
        ]
        
        self.logger.debug("DocumentTransformer инициализирован",
                         undefined_styles_count=len(self.undefined_style_ids))
    
    def create_transform_function(self):
        """
        Создает функцию трансформации для обработки неопределенных стилей
        
        Returns:
            Функция трансформации для mammoth
        """
        def transform_paragraph(element):
            """
            Трансформирует параграфы с неопределенными стилями
            
            Args:
                element: Элемент параграфа
                
            Returns:
                Трансформированный элемент
            """
            # Обработка параграфов с неопределенными стилями
            if element.style_id in self.undefined_style_ids:
                # Определяем тип стиля по ID
                if element.style_id in ['11', '24']:  # Заголовки
                    heading_level = 1 if element.style_id == '11' else 2
                    return element.copy(
                        style_id=f"Heading{heading_level}",
                        style_name=f"Заголовок №{heading_level}"
                    )
                elif element.style_id == 'a9':  # Подпись к таблице
                    return element.copy(
                        style_id="TableCaption",
                        style_name="Подпись к таблице"
                    )
                else:  # Остальные как обычный текст
                    return element.copy(
                        style_id="Normal",
                        style_name="Основной текст"
                    )
            
            # Обработка параграфов без имени стиля (None)
            if element.style_name == "None" and element.style_id:
                # Определяем по ID что это может быть
                if element.style_id in ['Style2', 'Style4', 'Style18']:
                    return element.copy(
                        style_id="Normal",
                        style_name="Основной текст"
                    )
            
            # Обработка параграфов без стиля вообще
            if not element.style_id and not element.style_name:
                return element.copy(
                    style_id="Normal",
                    style_name="Основной текст"
                )
            
            return element
        
        return mammoth.transforms.paragraph(transform_paragraph)
    
    def get_transform_options(self) -> dict:
        """
        Возвращает опции трансформации для mammoth
        
        Returns:
            Словарь с опциями трансформации
        """
        return {
            'transform_document': self.create_transform_function()
        }