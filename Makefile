up:
	docker compose -f api-gateway-service/deployment/docker-compose.yml --env-file .env --project-name api-gateway up -d --build
	docker compose -f user-service/deployment/docker-compose.yml --env-file .env --project-name user-service up -d --build
	docker compose -f doc-to-docx-converter/docker-compose.yml --project-name doc-to-docx-converter up -d --build
	docker compose -f docx-converter/docker-compose.yml --project-name docx-parser up -d --build
	docker compose -f llm-requester/docker-compose.yml --env-file .env  --project-name llm-requester up -d --build
	docker compose -f md-converter/deployment/docker-compose.yml --project-name html-to-markdown-converter up -d --build
	docker compose -f promt-builder/docker-compose.yml --env-file .env --project-name prompt-builder up -d --build
	docker compose -f report-generator/docker-compose.yml --project-name report-generator up -d --build
	docker compose -f tz-bot/deployment/docker-compose.yml --env-file .env --project_name tz-service up -d --build


stop:
	docker compose -f api-gateway-service/deployment/docker-compose.yml --env-file .env --project-name api-gateway stop

down:
	docker compose -f api-gateway-service/deployment/docker-compose.yml --env-file .env --project-name api-gateway down

up2:
	#docker compose -f docker-compose.yml --project-name common up -d
	docker compose -f api-gateway-service/deployment/docker-compose.yml --env-file .env --project-name api-gateway up -d --build

stop2:
	#docker compose -f docker-compose.yml --project-name common stop
	docker compose -f api-gateway-service/deployment/docker-compose.yml --env-file .env --project-name api-gateway stop

down2:
	#docker compose -f docker-compose.yml --project-name common down
	docker compose -f api-gateway-service/deployment/docker-compose.yml --env-file .env --project-name api-gateway down