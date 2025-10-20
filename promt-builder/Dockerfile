# syntax=docker/dockerfile:1

# Use a slim Python image; psycopg2-binary has wheels for this
FROM python:3.12-slim AS runtime

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    PIP_NO_CACHE_DIR=1 \
    PIP_DISABLE_PIP_VERSION_CHECK=1

WORKDIR /app

# Install Python deps
COPY requirements.txt /app/requirements.txt
RUN pip install --upgrade pip && \
    pip install -r /app/requirements.txt

# Copy service code
COPY . /app/PromptBuilder

EXPOSE 8000

# FastAPI app lives at PromptBuilder.app.main:app
CMD ["uvicorn", "PromptBuilder.app.main:app", "--host", "0.0.0.0", "--port", "8000"]
