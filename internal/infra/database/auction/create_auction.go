package auction

import (
	"context"
	"fullcycle-auction_go/configuration/logger"
	"fullcycle-auction_go/internal/entity/auction_entity"
	"fullcycle-auction_go/internal/internal_error"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap/zapcore"
)

type AuctionEntityMongo struct {
	Id          string                          `bson:"_id"`
	ProductName string                          `bson:"product_name"`
	Category    string                          `bson:"category"`
	Description string                          `bson:"description"`
	Condition   auction_entity.ProductCondition `bson:"condition"`
	Status      auction_entity.AuctionStatus    `bson:"status"`
	Timestamp   int64                           `bson:"timestamp"`
}
type AuctionRepository struct {
	Collection      *mongo.Collection
	auctionInterval time.Duration
}

func NewAuctionRepository(database *mongo.Database) *AuctionRepository {
	return &AuctionRepository{
		Collection:      database.Collection("auctions"),
		auctionInterval: getAuctionInterval(),
	}
}

func (ar *AuctionRepository) CreateAuction(
	ctx context.Context,
	auctionEntity *auction_entity.Auction) *internal_error.InternalError {
	auctionEntityMongo := &AuctionEntityMongo{
		Id:          auctionEntity.Id,
		ProductName: auctionEntity.ProductName,
		Category:    auctionEntity.Category,
		Description: auctionEntity.Description,
		Condition:   auctionEntity.Condition,
		Status:      auctionEntity.Status,
		Timestamp:   auctionEntity.Timestamp.Unix(),
	}
	_, err := ar.Collection.InsertOne(ctx, auctionEntityMongo)
	if err != nil {
		logger.Error("Error trying to insert auction", err)
		return internal_error.NewInternalServerError("Error trying to insert auction")
	}

	go ar.monitorAuction(ctx, auctionEntity)

	return nil
}

// monitorAuction inicia a monitoração do leilão até sua conclusão
func (ar *AuctionRepository) monitorAuction(ctx context.Context, auction *auction_entity.Auction) {
	timer := time.NewTimer(time.Until(auction.Timestamp.Add(ar.auctionInterval)))
	select {
	case <-timer.C:
		// Atualiza
		if err := ar.updateAuctionStatus(ctx, auction.Id, auction_entity.Completed); err != nil {
			logger.Error("Error trying to update auction status", err)
		}
		logger.Info("Auction completed after interval", zapcore.Field{Key: "auction_id", Type: zapcore.StringType, String: auction.Id})
		return
	case <-ctx.Done():
		return
	}
}

// updateAuctionStatus atualiza o status de um leilão no banco de dados
func (ar *AuctionRepository) updateAuctionStatus(ctx context.Context, auctionID string, status auction_entity.AuctionStatus) error {
	filter := bson.M{"_id": auctionID}
	update := bson.M{"$set": bson.M{"status": status}}

	_, err := ar.Collection.UpdateOne(ctx, filter, update)
	return err
}

// getAuctionInterval obtém o intervalo de tempo configurado para o leilão
func getAuctionInterval() time.Duration {
	auctionInterval := os.Getenv("AUCTION_INTERVAL")
	if auctionInterval == "" {
		return 5 * time.Minute // Valor padrão
	}

	duration, err := time.ParseDuration(auctionInterval)
	if err != nil {
		return 5 * time.Minute
	}
	return duration
}
