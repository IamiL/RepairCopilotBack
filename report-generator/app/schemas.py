from typing import List, Optional
from pydantic import BaseModel, Field, field_validator


class Instance(BaseModel):
    risks: str = Field(..., description="Риски (вынести в комментарии Word)")
    llm_id: str = Field(..., description="ID правила/нахождения (вынести в комментарии Word)")
    error_id: str
    priority: str = Field(..., description="low | medium | high")
    how_to_fix: str = Field(..., description="Как исправить")
    what_is_incorrect: str = Field(..., description="Что некорректно (якорь для комментария)")


class Section(BaseModel):
    part: str = Field(..., description="Название секции как в исходном ТЗ")
    # ВАЖНО: по умолчанию пустые списки и мягкая обработка null
    instances: List[Instance] = Field(default_factory=list)
    exists_in_doc: Optional[bool] = Field(default=None, description="Секция есть в исходном документе?")
    final_instance_ids: List[str] = Field(default_factory=list)
    initial_instance_ids: List[str] = Field(default_factory=list)

    # null → []
    @field_validator("instances", "final_instance_ids", "initial_instance_ids", mode="before")
    @classmethod
    def none_to_empty_list(cls, v):
        return [] if v is None else v


class ReportRequest(BaseModel):
    sections: List[Section] = Field(default_factory=list)
    doc_title: str = Field(..., description="Заголовок исходного документа")

    # null → []
    @field_validator("sections", mode="before")
    @classmethod
    def none_sections_to_empty(cls, v):
        return [] if v is None else v

    class Config:
        json_schema_extra = {
            "example": {
                "sections": [
                    {
                        "part": "7. Основные функции киосков:",
                        "instances": [
                            {
                                "risks": "Снижение воспринимаемой профессиональности документа...",
                                "llm_id": "E01A-1",
                                "error_id": "uuid-here",
                                "priority": "low",
                                "how_to_fix": "Удалить пробел перед точкой...",
                                "what_is_incorrect": "Лишний пробел перед точкой в конце пункта списка"
                            }
                        ],
                        "exists_in_doc": True,
                        "final_instance_ids": ["E01A-1"],
                        "initial_instance_ids": ["E01A-1"]
                    }
                ],
                "doc_title": "Внедрение информационного киоска для АО «Уральская Сталь»"
            }
        }
