package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/marcioaraujo/pos-auction/configuration/database/mongodb"
	"github.com/marcioaraujo/pos-auction/internal/infra/api/web/controller/auction_controller"
	"github.com/marcioaraujo/pos-auction/internal/infra/api/web/controller/bid_controller"
	"github.com/marcioaraujo/pos-auction/internal/infra/api/web/controller/user_controller"
	"github.com/marcioaraujo/pos-auction/internal/infra/database/auction"
	"github.com/marcioaraujo/pos-auction/internal/infra/database/bid"
	"github.com/marcioaraujo/pos-auction/internal/infra/database/user"
	"github.com/marcioaraujo/pos-auction/internal/usecase/auction_usecase"
	"github.com/marcioaraujo/pos-auction/internal/usecase/bid_usecase"
	"github.com/marcioaraujo/pos-auction/internal/usecase/user_usecase"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
)

func main() {
	ctx := context.Background()

	// Carregar as variáveis de ambiente
	if err := godotenv.Load("cmd/auction/.env"); err != nil {
		log.Fatal("Error loading .env file")
	}

	// Estabelecer a conexão com o banco de dados MongoDB
	databaseConnection, err := mongodb.NewMongoDBConnection(ctx)
	if err != nil {
		log.Fatal(err.Error())
		return
	}

	// Inicializar o roteador Gin
	router := gin.Default()

	// Inicializar os controllers
	userController, bidController, auctionController := initDependencies(databaseConnection)

	// Rotas
	router.GET("/auction", auctionController.FindAuctions)
	router.GET("/auction/:auctionId", auctionController.FindAuctionById)
	router.POST("/auction", auctionController.CreateAuction)
	router.GET("/auction/winner/:auctionId", auctionController.FindWinningBidByAuctionId)
	router.POST("/bid", bidController.CreateBid)
	router.GET("/bid/:auctionId", bidController.FindBidByAuctionId)
	router.GET("/user/:userId", userController.FindUserById)

	// Rodar a aplicação na porta 8080
	go monitorAuctions(databaseConnection) // Iniciar a goroutine para monitorar leilões

	router.Run(":8080")
}

// Função que inicializa as dependências da aplicação
func initDependencies(database *mongo.Database) (
	userController *user_controller.UserController,
	bidController *bid_controller.BidController,
	auctionController *auction_controller.AuctionController) {

	auctionRepository := auction.NewAuctionRepository(database)
	bidRepository := bid.NewBidRepository(database, auctionRepository)
	userRepository := user.NewUserRepository(database)

	userController = user_controller.NewUserController(
		user_usecase.NewUserUseCase(userRepository))
	auctionController = auction_controller.NewAuctionController(
		auction_usecase.NewAuctionUseCase(auctionRepository, bidRepository))
	bidController = bid_controller.NewBidController(bid_usecase.NewBidUseCase(bidRepository))

	return
}

// Função para monitorar e fechar automaticamente os leilões
func monitorAuctions(database *mongo.Database) {
	// Leitura da variável de duração de leilão do ambiente
	duration, err := strconv.Atoi(os.Getenv("AUCTION_DURATION"))
	if err != nil {
		duration = 60 // Valor padrão de 60 minutos se a variável não for definida
	}

	// Monitorar leilões para fechá-los quando o tempo expirar
	for {
		// Buscar todos os leilões abertos
		auctions, err := auction.NewAuctionRepository(database).FindOpenAuctions()
		if err != nil {
			log.Printf("Error fetching auctions: %v", err)
			time.Sleep(30 * time.Second) // Aguardar antes de tentar novamente
			continue
		}

		// Verificar e fechar leilões que passaram do tempo
		for _, auction := range auctions {
			if time.Now().After(time.Unix(auction.Timestamp, 0).Add(time.Duration(duration) * time.Minute)) {
				err := auction.NewAuctionRepository(database).CloseAuction(context.Background(), auction)
				if err != nil {
					log.Printf("Error closing auction %s: %v", auction.Id, err)
				} else {
					log.Printf("Auction %s closed automatically", auction.Id)
				}
			}
		}

		// Aguardar antes de verificar novamente
		time.Sleep(1 * time.Minute)
	}
}
