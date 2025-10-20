from fastapi import APIRouter, HTTPException
from fastapi.responses import JSONResponse
from pydantic import BaseModel
from app.services.converter_service import ConverterService

router = APIRouter(tags=["converter"])


class ConvertRequest(BaseModel):
    html_text: str


@router.post("/convert")
async def convert_html(request: ConvertRequest):
    """
    Конвертирует HTML текст в Markdown
    """
    if not request.html_text:
        raise HTTPException(status_code=400, detail="HTML текст не предоставлен")
    
    try:
        converter_service = ConverterService()
        result = await converter_service.convert_html_text(request.html_text)
        
        return JSONResponse(
            status_code=200,
            content={
                "message": "HTML успешно конвертирован",
                "markdown": result.get("markdown_result", ""),
                "html_with_ids": result.get("html_with_ids", ""),
                "mappings": result.get("mappings", []),
                "original_html_length": result.get("original_html_length", 0),
                "status": result.get("status", "converted")
            }
        )
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Ошибка при конвертации: {str(e)}")