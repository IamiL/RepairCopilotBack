from fastapi import FastAPI
from fastapi.responses import StreamingResponse, JSONResponse
from fastapi.middleware.cors import CORSMiddleware
from .schemas import ReportRequest
from .report import build_report
import io
from datetime import datetime

app = FastAPI(
    title="Генератор отчёта об ошибках ТЗ (.docx)",
    description="HTTP-сервис: принимает JSON и возвращает Word-документ с отчётом и комментариями (Review).",
    version="1.0.1",
)

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

@app.get("/health")
def health():
    return {"status": "ok"}

@app.post("/report", summary="Сгенерировать .docx отчёт из JSON", response_description="Возвращает файл .docx")
def create_report(req: ReportRequest):
    try:
        data = build_report(req)
        if not data:
            return JSONResponse(status_code=500, content={"error": "empty document generated"})
    except Exception as e:
        return JSONResponse(status_code=500, content={"error": str(e)})

    filename = f"report_{datetime.now().strftime('%Y%m%d_%H%M%S')}.docx"
    return StreamingResponse(
        io.BytesIO(data),
        media_type="application/vnd.openxmlformats-officedocument.wordprocessingml.document",
        headers={"Content-Disposition": f'attachment; filename="{filename}"'}
    )
