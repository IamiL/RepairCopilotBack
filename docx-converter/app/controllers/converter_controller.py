from fastapi import APIRouter, File, UploadFile, HTTPException, Query
from fastapi.responses import JSONResponse, HTMLResponse
from app.services.converter_service import ConverterService
from app.utils.logger import get_logger

router = APIRouter(tags=["converter"])
logger = get_logger(__name__)


@router.post("/convert")
async def convert_file(
    file: UploadFile = File(...),
    format: str = Query("json", description="Формат ответа: json или html")
):
    """
    Конвертирует DOCX файл в HTML
    
    Parameters:
    - file: DOCX файл для конвертации
    - format: Формат ответа - 'json' (по умолчанию) или 'html'
    
    Returns:
    - При format=json: JSON с информацией о конвертации
    - При format=html: HTML страница с результатом конвертации
    """
    request_id = id(file)
    logger.info(
        "Начало обработки запроса на конвертацию",
        request_id=request_id,
        filename=file.filename,
        content_type=file.content_type,
        response_format=format
    )
    
    try:
        converter_service = ConverterService()
        result = await converter_service.convert_file(file)
        
        logger.info(
            "Конвертация успешно завершена",
            request_id=request_id,
            filename=result["original_filename"],
            file_size=result["file_size"],
            warnings_count=result["warnings_count"],
            errors_count=result["errors_count"],
            status=result["status"]
        )
        
        if format.lower() == "html":
            logger.debug("Возврат результата в HTML формате", request_id=request_id)
            return HTMLResponse(
                content=result["complete_html"],
                status_code=200
            )
        
        logger.debug("Возврат результата в JSON формате", request_id=request_id)
        return JSONResponse(
            status_code=200,
            content={
                "message": "Файл успешно конвертирован",
                "original_filename": result["original_filename"],
                "file_size": result["file_size"],
                "html_content": result["html_content"],
                "complete_html": result["complete_html"],
                "conversion_warnings": result["conversion_warnings"],
                "conversion_errors": result["conversion_errors"],
                "style_analysis": result["style_analysis"],
                "warning_summary": result["warning_summary"],
                "warnings_count": result["warnings_count"],
                "errors_count": result["errors_count"],
                "status": result["status"]
            }
        )
        
    except HTTPException as he:
        logger.warning(
            "HTTP исключение при обработке запроса",
            request_id=request_id,
            status_code=he.status_code,
            detail=he.detail
        )
        raise
    except Exception as e:
        logger.exception(
            "Неожиданная ошибка при обработке запроса",
            request_id=request_id,
            error_type=type(e).__name__,
            error_message=str(e)
        )
        raise HTTPException(
            status_code=500, 
            detail=f"Внутренняя ошибка сервера: {str(e)}"
        )