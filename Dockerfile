FROM golang:1.20

WORKDIR /app

# Copiar dependências e baixar
COPY go.mod ./
COPY go.sum ./
RUN go mod download

# Copiar todo o código para o container
COPY . .

# Construir o binário do Go
RUN go build -o /app/auction cmd/auction/main.go

# Expor a porta necessária
EXPOSE 8080

# Configurar o ENTRYPOINT para rodar a aplicação
ENTRYPOINT ["/app/auction"]f