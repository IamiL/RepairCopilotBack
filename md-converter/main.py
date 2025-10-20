import uvicorn
from fastapi import FastAPI
from app.controllers.converter_controller import router as converter_router


def create_app() -> FastAPI:
    app = FastAPI(
        title="Document Converter API",
        description="API для конвертации документов",
        version="1.0.0"
    )
    
    app.include_router(converter_router, prefix="/api/v1")
    
    return app


if __name__ == "__main__":
    app = create_app()
    uvicorn.run(app, host="0.0.0.0", port=8000)