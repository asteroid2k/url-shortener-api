package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"

	// "go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Url Model
type Url struct {
	// ID     primitive.ObjectID `bson:"_id,omitempty"`
	Full   string `bson:"full,omitempty"`
	Short  string `bson:"short,omitempty"`
	Clicks int    `bson:"clicks,omitempty"`
}

func NewUrl(full, short string, clicks int) Url {
	u := Url{
		Full:   full,
		Short:  short,
		Clicks: clicks,
	}
	return u
}

var (
	mongoURI = "mongodb://localhost:27017"
)

func main() {
	// DATABASE
	// connect Database
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	// context with timeout
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(ctx)
	//get collection handle
	urlCollection := client.Database("url_shortener").Collection("urlz")

	// Fiber
	app := fiber.New()

	//Routes
	// Landing
	app.Get("/", func(c *fiber.Ctx) error {
		var allUrls []Url
		cur, err := urlCollection.Find(context.Background(), bson.D{})
		defer cur.Close(context.Background())

		if err = cur.All(ctx, &allUrls); err != nil {
			log.Fatal(err)
		}

		return c.JSON(allUrls)

	})

	// add url
	app.Post("/", func(c *fiber.Ctx) error {
		uri := struct {
			Full string `json:"full"`
			Sub  string `json:"subdomain"`
		}{}
		if err := c.BodyParser(&uri); err != nil {
			return err
		}
		short := uri.Sub + "-" + genId(5)
		// create new url struct
		sUrl := Url{
			Full:   uri.Full,
			Short:  short,
			Clicks: 0,
		}

		_, err = urlCollection.InsertOne(ctx, sUrl)
		if err != nil {
			log.Fatal(err)
		}

		if err := c.Status(200).JSON(&fiber.Map{
			"message": "Url created",
		}); err != nil {
			c.Status(500).JSON(&fiber.Map{
				"message": "DB error",
				"body":    err,
			})
		}
		return nil
	})

	// Get full url
	app.Get("/:shortUrl", func(c *fiber.Ctx) error {
		shortUrl := c.Params("shortUrl")
		filter := bson.D{{"short", shortUrl}}
		var url Url
		err := urlCollection.FindOne(ctx, filter).Decode(&url)
		if err != nil {
			fmt.Println(err)
			return c.Status(400).JSON(fiber.Map{
				"message": "Url not found",
			})
		}

		// increment clicks or short url
		clicks := url.Clicks + 1
		update := bson.D{{"$set", bson.D{{"clicks", clicks}}}}
		res, err := urlCollection.UpdateOne(ctx, filter, update)
		if err != nil {
			log.Fatal(err)
		}

		return c.Status(200).JSON(fiber.Map{
			"fullUrl": url.Full,
			"res":     res,
		})
	})

	// Start server
	app.Listen(":3000")
}

func genId(n int) string {
	rand.Seed(time.Now().UnixNano())
	var letters = []rune("abcdefghijklmnopqrstuvwxyz.!@#$^=-ABCDEFGHIJKLMNOPQRSTUVWXY0123456789")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
