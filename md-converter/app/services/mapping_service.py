from typing import Dict, List, Any
from dataclasses import dataclass
from bs4 import BeautifulSoup, Tag
from markdownify import markdownify as md
import uuid


@dataclass
class HtmlToMarkdownMapping:
    """Структура для хранения связи HTML блока и строк Markdown"""
    html_element_id: str
    html_content: str
    html_tag: str
    markdown_line_start: int
    markdown_line_end: int
    markdown_content: str


class MappingService:
    """Сервис для создания маппинга между HTML элементами и строками Markdown"""
    
    def __init__(self):
        self.mappings: List[HtmlToMarkdownMapping] = []
    
    def convert_with_mapping(self, html_text: str) -> Dict[str, Any]:
        """
        Конвертирует HTML в Markdown с созданием маппинга
        
        Args:
            html_text: HTML текст для конвертации
            
        Returns:
            Dict с результатом конвертации и маппингом
        """
        try:
            # Парсим HTML
            soup = BeautifulSoup(html_text, 'html.parser')
            
            # Добавляем уникальные ID ко всем элементам
            self._add_unique_ids(soup)
            
            # Конвертируем в Markdown
            markdown_result = md(str(soup), 
                               heading_style="ATX",
                               bullets="-",
                               strip=['script', 'style'])
            
            # Создаем маппинг с оригинальным markdown
            self._create_mapping(soup, markdown_result)
            
            # Добавляем нумерацию строк только в итоговый результат
            numbered_markdown = self._add_line_numbers(markdown_result)
            
            return {
                "original_html_length": len(html_text),
                "markdown_result": numbered_markdown.strip(),
                "html_with_ids": str(soup),
                "mappings": [self._mapping_to_dict(m) for m in self.mappings],
                "status": "converted"
            }
            
        except Exception as e:
            return {
                "original_html_length": len(html_text),
                "markdown_result": "",
                "mappings": [],
                "status": "error",
                "error": str(e)
            }
    
    def _add_line_numbers(self, markdown_text: str) -> str:
        """Добавляет номера строк в формате [xxx] к каждой строке markdown"""
        lines = markdown_text.split('\n')
        numbered_lines = []
        
        for i, line in enumerate(lines, 1):
            line_number = f"[{i:03d}]"
            numbered_lines.append(f"{line_number} {line}")
        
        return '\n'.join(numbered_lines)
    
    def _add_unique_ids(self, soup: BeautifulSoup) -> None:
        """Добавляет уникальные ID только к HTML элементам первого уровня"""
        for element in soup.children:  # Находим только элементы первого уровня
            if isinstance(element, Tag):
                element['data-mapping-id'] = str(uuid.uuid4())
    
    def _create_mapping(self, soup: BeautifulSoup, markdown_text: str) -> None:
        """Создает маппинг между HTML элементами и строками Markdown"""
        self.mappings = []
        markdown_lines = markdown_text.split('\n')
        current_line = 0
        
        # Обходим только элементы первого уровня в порядке их появления
        for element in soup.children:
            if isinstance(element, Tag) and element.get('data-mapping-id'):
                # Конвертируем отдельный элемент в markdown
                element_markdown = md(str(element), 
                                    heading_style="ATX",
                                    bullets="-",
                                    strip=['script', 'style']).strip()
                
                # Создаем маппинг даже для пустых элементов
                if element_markdown:
                    # Находим этот контент в общем markdown
                    line_start, line_end = self._find_content_lines(
                        markdown_lines, element_markdown, current_line
                    )
                    
                    if line_start != -1:
                        mapping = HtmlToMarkdownMapping(
                            html_element_id=element.get('data-mapping-id'),
                            html_content=str(element),
                            html_tag=element.name,
                            markdown_line_start=line_start + 1,  # 1-based indexing
                            markdown_line_end=line_end + 1,
                            markdown_content=element_markdown
                        )
                        self.mappings.append(mapping)
                        current_line = line_end
                else:
                    # Для пустых элементов создаем маппинг без привязки к строкам markdown
                    mapping = HtmlToMarkdownMapping(
                        html_element_id=element.get('data-mapping-id'),
                        html_content=str(element),
                        html_tag=element.name,
                        markdown_line_start=-1,  # Указываем, что строки нет
                        markdown_line_end=-1,
                        markdown_content=""
                    )
                    self.mappings.append(mapping)
    
    def _find_content_lines(self, markdown_lines: List[str], content: str, start_from: int = 0) -> tuple:
        """Находит диапазон строк, содержащих указанный контент"""
        content_lines = content.split('\n')
        
        for i in range(start_from, len(markdown_lines)):
            # Проверяем, начинается ли здесь искомый контент
            match = True
            for j, content_line in enumerate(content_lines):
                if i + j >= len(markdown_lines) or markdown_lines[i + j].strip() != content_line.strip():
                    match = False
                    break
            
            if match:
                return i, i + len(content_lines) - 1
        
        return -1, -1
    
    def _mapping_to_dict(self, mapping: HtmlToMarkdownMapping) -> Dict[str, Any]:
        """Конвертирует маппинг в словарь для JSON ответа"""
        return {
            "html_element_id": mapping.html_element_id,
            "html_tag": mapping.html_tag,
            "html_content": mapping.html_content,
            "markdown_line_start": mapping.markdown_line_start,
            "markdown_line_end": mapping.markdown_line_end,
            "markdown_content": mapping.markdown_content
        }
    
    def find_html_by_line(self, line_number: int) -> Dict[str, Any]:
        """Находит HTML блок по номеру строки Markdown"""
        for mapping in self.mappings:
            if mapping.markdown_line_start <= line_number <= mapping.markdown_line_end:
                return self._mapping_to_dict(mapping)
        return {}