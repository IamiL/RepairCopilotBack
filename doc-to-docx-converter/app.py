import os
import subprocess
import tempfile
import shutil
from pathlib import Path
from urllib.parse import quote  # Добавлен импорт
from fastapi import FastAPI, File, UploadFile, HTTPException
from fastapi.responses import FileResponse, JSONResponse
from fastapi.middleware.cors import CORSMiddleware
import uvicorn
from typing import Optional
import logging

# Настройка логирования
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

app = FastAPI(title="DOC to DOCX Converter", version="1.0.0")

# Настройка CORS
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Создание директорий для временных файлов
UPLOAD_DIR = Path("/tmp/uploads")
OUTPUT_DIR = Path("/tmp/outputs")
UPLOAD_DIR.mkdir(exist_ok=True)
OUTPUT_DIR.mkdir(exist_ok=True)


def cleanup_existing_files(input_filename: str):
    """
    Удаляет существующие файлы с таким же именем в директориях uploads и outputs
    """
    try:
        # Очистка input файла
        input_path = UPLOAD_DIR / input_filename
        if input_path.exists():
            input_path.unlink()
            logger.info(f"Removed existing input file: {input_path}")
        
        # Очистка output файла
        output_filename = input_filename.replace('.doc', '.docx')
        output_path = OUTPUT_DIR / output_filename
        if output_path.exists():
            output_path.unlink()
            logger.info(f"Removed existing output file: {output_path}")
            
        # Очистка файла созданного LibreOffice (если отличается от желаемого)
        input_name_without_ext = input_filename.replace('.doc', '')
        libreoffice_output = OUTPUT_DIR / f"{input_name_without_ext}.docx"
        if libreoffice_output.exists() and libreoffice_output != output_path:
            libreoffice_output.unlink()
            logger.info(f"Removed existing LibreOffice output file: {libreoffice_output}")
            
    except Exception as e:
        logger.warning(f"Error during cleanup: {str(e)}")


def convert_doc_to_docx(input_path: Path, output_path: Path) -> bool:
    """
    Конвертирует DOC файл в DOCX используя LibreOffice
    """
    try:
        # Команда для конвертации через LibreOffice
        cmd = [
            "libreoffice",
            "--headless",
            "--convert-to",
            "docx",
            "--outdir",
            str(output_path.parent),
            str(input_path)
        ]

        logger.info(f"Executing command: {' '.join(cmd)}")

        # Выполнение команды
        result = subprocess.run(
            cmd,
            capture_output=True,
            text=True,
            timeout=30
        )

        if result.returncode != 0:
            logger.error(f"Conversion failed: {result.stderr}")
            return False

        logger.info(f"Conversion successful: {result.stdout}")
        return True

    except subprocess.TimeoutExpired:
        logger.error("Conversion timeout")
        return False
    except Exception as e:
        logger.error(f"Conversion error: {str(e)}")
        return False


@app.get("/")
async def root():
    """Главная страница с информацией о сервисе"""
    return {
        "service": "DOC to DOCX Converter",
        "version": "1.0.0",
        "endpoints": {
            "/convert": "POST - конвертировать DOC в DOCX",
            "/health": "GET - проверка состояния сервиса",
            "/docs": "GET - документация API (Swagger UI)"
        }
    }


@app.get("/health")
async def health_check():
    """Проверка состояния сервиса"""
    try:
        # Проверка доступности LibreOffice
        result = subprocess.run(
            ["libreoffice", "--version"],
            capture_output=True,
            text=True,
            timeout=5
        )

        if result.returncode == 0:
            return {
                "status": "healthy",
                "libreoffice": "available",
                "version": result.stdout.strip()
            }
        else:
            return JSONResponse(
                status_code=503,
                content={"status": "unhealthy", "error": "LibreOffice not available"}
            )
    except Exception as e:
        return JSONResponse(
            status_code=503,
            content={"status": "unhealthy", "error": str(e)}
        )


@app.post("/convert")
async def convert_document(
        file: UploadFile = File(...),
        output_filename: Optional[str] = None
):
    """
    Конвертирует загруженный DOC файл в DOCX

    Parameters:
    - file: DOC файл для конвертации
    - output_filename: имя выходного файла (опционально)
    """

    # Проверка расширения файла
    if not file.filename.lower().endswith('.doc'):
        raise HTTPException(
            status_code=400,
            detail="File must have .doc extension"
        )

    # Генерация временных путей
    temp_input = UPLOAD_DIR / file.filename
    temp_output_name = output_filename or file.filename.replace('.doc', '.docx')
    temp_output = OUTPUT_DIR / temp_output_name

    try:
        # Очистка существующих файлов перед обработкой
        cleanup_existing_files(file.filename)
        
        # Сохранение загруженного файла
        logger.info(f"Saving uploaded file: {file.filename}")
        with open(temp_input, "wb") as buffer:
            shutil.copyfileobj(file.file, buffer)

        # Конвертация
        logger.info(f"Converting {temp_input} to {temp_output}")
        success = convert_doc_to_docx(temp_input, temp_output)

        if not success:
            raise HTTPException(
                status_code=500,
                detail="Conversion failed"
            )

        # LibreOffice создает файл с тем же именем, но с расширением .docx
        # Определяем правильное имя выходного файла
        input_name_without_ext = temp_input.name.replace('.doc', '')
        expected_output = OUTPUT_DIR / f"{input_name_without_ext}.docx"

        if not expected_output.exists():
            raise HTTPException(
                status_code=500,
                detail="Output file not created"
            )

        # Переименование если нужно
        if expected_output != temp_output:
            shutil.move(str(expected_output), str(temp_output))

        # Правильное формирование заголовка Content-Disposition для кириллицы
        encoded_filename = quote(temp_output_name.encode('utf-8'))

        # Возврат конвертированного файла
        return FileResponse(
            path=temp_output,
            media_type="application/vnd.openxmlformats-officedocument.wordprocessingml.document",
            filename=temp_output_name,
            headers={
                "Content-Disposition": f"attachment; filename*=UTF-8''{encoded_filename}"
            }
        )

    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Unexpected error: {str(e)}")
        raise HTTPException(
            status_code=500,
            detail=f"Internal server error: {str(e)}"
        )
    finally:
        # Очистка временных файлов
        if temp_input.exists():
            temp_input.unlink()
        # Выходной файл будет удален после отправки клиенту


@app.post("/convert-batch")
async def convert_batch(files: list[UploadFile] = File(...)):
    """
    Конвертирует несколько DOC файлов в DOCX
    """
    results = []

    for file in files:
        if not file.filename.lower().endswith('.doc'):
            results.append({
                "filename": file.filename,
                "status": "error",
                "message": "File must have .doc extension"
            })
            continue

        temp_input = UPLOAD_DIR / file.filename
        temp_output_name = file.filename.replace('.doc', '.docx')
        temp_output = OUTPUT_DIR / temp_output_name

        try:
            # Очистка существующих файлов перед обработкой
            cleanup_existing_files(file.filename)
            
            # Сохранение файла
            with open(temp_input, "wb") as buffer:
                shutil.copyfileobj(file.file, buffer)

            # Конвертация
            success = convert_doc_to_docx(temp_input, temp_output)

            if success:
                input_name_without_ext = temp_input.name.replace('.doc', '')
                expected_output = OUTPUT_DIR / f"{input_name_without_ext}.docx"

                if expected_output.exists():
                    if expected_output != temp_output:
                        shutil.move(str(expected_output), str(temp_output))

                    results.append({
                        "filename": file.filename,
                        "status": "success",
                        "output": temp_output_name
                    })
                else:
                    results.append({
                        "filename": file.filename,
                        "status": "error",
                        "message": "Output file not created"
                    })
            else:
                results.append({
                    "filename": file.filename,
                    "status": "error",
                    "message": "Conversion failed"
                })

        except Exception as e:
            results.append({
                "filename": file.filename,
                "status": "error",
                "message": str(e)
            })
        finally:
            if temp_input.exists():
                temp_input.unlink()

    return {"results": results}


if __name__ == "__main__":
    uvicorn.run(app, host="0.0.0.0", port=8000)