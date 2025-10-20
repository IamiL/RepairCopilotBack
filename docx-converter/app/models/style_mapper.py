from typing import Dict, List, Pattern
import re
from app.utils.logger import get_logger


class StyleMapper:
    """
    Класс для работы с маппингом стилей и обработки нераспознанных стилей
    """
    
    def __init__(self):
        self.logger = get_logger(__name__)
        self.russian_headings = [
            'Заголовок №1', 'Заголовок №2', 'Заголовок №3', 
            'Заголовок №4', 'Заголовок №5', 'Заголовок №6'
        ]
        self.text_styles = [
            'Основной текст', 'Основной текст (2)', 'Основной текст (3)',
            'Основной текст1', 'Обычный текст', 'Normal'
        ]
        self.special_styles = {
            'Подпись к таблице': 'table-caption',
            'Другое': 'other',
            'None': 'default'
        }
        
        self.logger.debug("StyleMapper инициализирован",
                         headings_count=len(self.russian_headings),
                         text_styles_count=len(self.text_styles),
                         special_styles_count=len(self.special_styles))
    
    def create_style_mapping(self) -> str:
        """
        Создает полную карту стилей для Mammoth
        
        Returns:
            Строка с маппингом стилей
        """
        mappings = []
        
        # Заголовки
        for i, heading in enumerate(self.russian_headings, 1):
            mappings.append(f"p[style-name='{heading}'] => h{i}:fresh")
        
        # Основной текст
        for text_style in self.text_styles:
            mappings.append(f"p[style-name='{text_style}'] => p:fresh")
        
        # Специальные стили
        for style_name, css_class in self.special_styles.items():
            if css_class == 'default':
                mappings.append(f"p[style-name='{style_name}'] => p:fresh")
            else:
                mappings.append(f"p[style-name='{style_name}'] => p.{css_class}:fresh")
        
        # Обработка стилей по ID - используем style-name вместо style-id
        # Mammoth не поддерживает синтаксис style-id, только style-name
        # Эти стили будут обрабатываться через document transformer
        mappings.extend([
            "p[style-name='Основной текст'] => p:fresh",
            "p[style-name='Заголовок №1'] => h1:fresh",
            "p[style-name='Заголовок №2'] => h2:fresh",
            "p[style-name='Подпись к таблице'] => p.table-caption:fresh",
            "p[style-name='TableCaption'] => p.table-caption:fresh",
            "p[style-name='Heading1'] => h1:fresh",
            "p[style-name='Heading2'] => h2:fresh",
            "p[style-name='Normal'] => p:fresh"
        ])
        
        style_map = "\n".join(mappings)
        self.logger.debug("Создана карта стилей", 
                         mappings_count=len(mappings))
        return style_map
    
    def analyze_style_warnings(self, warnings: List[str]) -> Dict[str, List[str]]:
        """
        Анализирует предупреждения о стилях и категоризирует их
        
        Args:
            warnings: Список предупреждений от Mammoth
            
        Returns:
            Словарь с категоризированными предупреждениями
        """
        self.logger.debug("Анализ предупреждений о стилях", 
                         warnings_count=len(warnings))
        
        categorized = {
            'undefined_styles': [],
            'unrecognized_styles': [],
            'missing_elements': [],
            'table_formatting_ignored': [],
            'other_warnings': []
        }
        
        for warning in warnings:
            if 'was referenced but not defined' in warning:
                categorized['undefined_styles'].append(warning)
            elif 'Unrecognised paragraph style' in warning:
                categorized['unrecognized_styles'].append(warning)
            elif 'unrecognised element was ignored' in warning:
                if 'w:tblPrEx' in warning:
                    categorized['table_formatting_ignored'].append(warning)
                else:
                    categorized['missing_elements'].append(warning)
            else:
                categorized['other_warnings'].append(warning)
        
        self.logger.debug("Анализ завершен",
                         undefined_styles=len(categorized['undefined_styles']),
                         unrecognized_styles=len(categorized['unrecognized_styles']),
                         missing_elements=len(categorized['missing_elements']))
        
        return categorized
    
    def extract_style_info(self, warning: str) -> Dict[str, str]:
        """
        Извлекает информацию о стиле из предупреждения
        
        Args:
            warning: Текст предупреждения
            
        Returns:
            Словарь с информацией о стиле
        """
        # Паттерн для извлечения имени стиля и ID
        style_pattern = r"Unrecognised paragraph style: (.+?) \(Style ID: (.+?)\)"
        match = re.search(style_pattern, warning)
        
        if match:
            return {
                'style_name': match.group(1),
                'style_id': match.group(2)
            }
        
        # Паттерн для неопределенных стилей
        undefined_pattern = r"Paragraph style with ID (.+?) was referenced"
        match = re.search(undefined_pattern, warning)
        
        if match:
            return {
                'style_id': match.group(1),
                'style_name': 'Unknown'
            }
        
        return {'style_name': 'Unknown', 'style_id': 'Unknown'}
    
    def get_warning_summary(self, style_analysis: Dict[str, List[str]]) -> str:
        """
        Создает краткое описание предупреждений для пользователя
        
        Args:
            style_analysis: Результат анализа предупреждений
            
        Returns:
            Строка с описанием предупреждений
        """
        summary_parts = []
        
        if style_analysis['undefined_styles']:
            count = len(style_analysis['undefined_styles'])
            summary_parts.append(f"Неопределенные стили: {count}")
        
        if style_analysis['unrecognized_styles']:
            count = len(style_analysis['unrecognized_styles'])
            summary_parts.append(f"Нераспознанные стили: {count}")
        
        if style_analysis['table_formatting_ignored']:
            summary_parts.append("Форматирование таблиц проигнорировано (ожидаемо)")
            
        if style_analysis['missing_elements']:
            count = len(style_analysis['missing_elements'])
            summary_parts.append(f"Пропущенные элементы: {count}")
        
        if not summary_parts:
            return "Все стили обработаны успешно"
            
        return "; ".join(summary_parts)