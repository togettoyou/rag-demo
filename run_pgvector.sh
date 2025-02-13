docker run -d --name pgvector17 \
  -e POSTGRES_USER=pgvector \
  -e POSTGRES_PASSWORD=pgvector \
  -e POSTGRES_DB=llm-test \
  -v pgvector_data:/var/lib/postgresql/data \
  -p 5432:5432 \
  pgvector/pgvector:pg17
