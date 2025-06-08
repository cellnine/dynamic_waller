package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	// "net/url"
	// "crypto/sha256"
	// "encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	//"github.com/go-delve/delve/service"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	// smithyendpoints "github.com/aws/smithy-go/endpoints"
)

var (
	db       *gorm.DB
	rdb      *redis.Client
	s3Client *s3.Client
	ctx      = context.Background()
)

type Wallpaper struct {
	ID        string    `gorm:"primaryKey" json:"id"`
	Status    string    `json:"status"`
	LightImg  string    `json:"-"`
	DarkImg   string    `json:"-"`
	FinalURL  string    `json:"final_url"`
	CreatedAt time.Time `json:"created_at"`
}

func main() {
	dsn := "host=db user=user password=password dbname=wallpapers_db port=5432 sslmode=disable"

	log.Println("----------- LOADING ENVIRONMENT -----------")
	log.Printf("R2_ACCOUNT_ID: [%s]", os.Getenv("R2_ACCOUNT_ID"))
	log.Printf("R2_ACCESS_KEY_ID: [%s]", os.Getenv("R2_ACCESS_KEY_ID"))
	log.Printf("R2_SECRET_ACCESS_KEY is set: %v", os.Getenv("R2_SECRET_ACCESS_KEY") != "")
	log.Printf("R2_BUCKET_NAME: [%s]", os.Getenv("R2_BUCKET_NAME"))
	log.Printf("R2_PUBLIC_URL: [%s]", os.Getenv("R2_PUBLIC_URL"))
	log.Println("-----------------------------------------")

	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	db.AutoMigrate(&Wallpaper{})

	rdb = redis.NewClient(&redis.Options{
		Addr: "redis:6379",
	})

	accountID := os.Getenv("R2_ACCOUNT_ID")
	accessKeyID := os.Getenv("R2_ACCESS_KEY_ID")
	secretAccessKey := os.Getenv("R2_SECRET_ACCESS_KEY")

	r2Resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL: fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountID),
		}, nil
	})

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithEndpointResolverWithOptions(r2Resolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, "")),
		config.WithRegion("auto"),
		// config.WithClientLogMode(aws.LogSigning),
	)
	if err != nil {
		log.Fatalf("Unable to load SDK config, %v", err)
	}

	s3Client = s3.NewFromConfig(cfg)

	if len(os.Args) > 1 && os.Args[1] == "worker" {
		runWorker()
	} else {
		db.AutoMigrate(&Wallpaper{})
		runAPI()
	}
}

func runAPI() {
	router := gin.Default()
	router.StaticFile("/", "./static/index.html")
	router.Static("/static", "./static")

	api := router.Group("api")
	{
		api.POST("/create", createWallpaper)
		api.GET("/status/:id", getStatus)
		api.GET("/gallery", getGallery)
	}

	log.Println("Starting API server on port 8080")
	if err := router.Run(":8080"); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}

func createWallpaper(c *gin.Context) {
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid form data"})
		return
	}

	lightFiles := form.File["light"]
	darkFiles := form.File["dark"]
	if len(lightFiles) == 0 || len(darkFiles) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Both light and dark images are required"})
		return
	}
	lightFile := lightFiles[0]
	darkFile := darkFiles[0]

	jobID := uuid.New().String()
	tmpDir := filepath.Join("/tmp", jobID)
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create temp directory"})
		return
	}

	lightPath := filepath.Join(tmpDir, "light"+filepath.Ext(lightFile.Filename))
	darkPath := filepath.Join(tmpDir, "dark"+filepath.Ext(darkFile.Filename))

	if err := c.SaveUploadedFile(lightFile, lightPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not save light image"})
		return
	}
	if err := c.SaveUploadedFile(darkFile, darkPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not save dark image"})
		return
	}

	wallpaper := Wallpaper{
		ID:        jobID,
		Status:    "pending",
		LightImg:  lightPath,
		DarkImg:   darkPath,
		CreatedAt: time.Now(),
	}

	if err := db.Create(&wallpaper).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database record could not be ccreated"})
		return
	}

	if err := rdb.LPush(ctx, "wallpaper_jobs", jobID).Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Job could not be queued."})
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": jobID})
}

func getStatus(c *gin.Context) {
	id := c.Param("id")
	var wallpaper Wallpaper
	if err := db.First(&wallpaper, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Wallpaper not found"})
		return
	}
	c.JSON(http.StatusOK, wallpaper)
}

func getGallery(c *gin.Context) {
	var wallpapers []Wallpaper

	db.Where("status = ?", "completed").Order("created_at desc").Find(&wallpapers)
	c.JSON(http.StatusOK, wallpapers)
}

func runWorker() {
	log.Println("Starting worker process...")
	for {
		result, err := rdb.BRPop(ctx, 0, "wallpaper_jobs").Result()
		if err != nil {
			log.Printf("Error pulling job from Redis: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}

		jobID := result[1]
		log.Printf("Processing job %s", jobID)

		var wallpaper Wallpaper
		if err := db.First(&wallpaper, "id = ?", jobID).Error; err != nil {
			log.Printf("Error finding job %s in DB: %v", jobID, err)
			continue
		}

		db.Model(&wallpaper).Update("Status", "processing")

		tmpDir := filepath.Dir(wallpaper.LightImg)
		finalPath := filepath.Join(tmpDir, "dynamic.heic")

		xmpContent := `<?xpacket?><x:xmpmeta xmlns:x="adobe:ns:meta"><rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns"><rdf:Description xmlns:apple_desktop="http://ns.apple.com/namespace/1.0" apple_desktop:apr="YnBsaXN0MDDSAQMCBFFsEAFRZBAACA0TEQ/REMOVE/8BAQAAAAAAAAAFAAAAAAAAAAAAAAAAAAAAFQ=="/></rdf:RDF></x:xmpmeta>`

		baseName := filepath.Base(wallpaper.LightImg)
		ext := filepath.Ext(baseName)
		xmpFileName := baseName[0:len(baseName)-len(ext)] + ".xmp"
		xmpPath := filepath.Join(tmpDir, xmpFileName)

		if err := os.WriteFile(xmpPath, []byte(xmpContent), 0644); err != nil {
			log.Printf("Failed to create xmp file for job %s: %v", jobID, err)
			db.Model(&wallpaper).Update("Status", "failed")
			continue
		}

		cmdExiv := exec.Command("exiv2", "-iX", "in", wallpaper.LightImg)
		if output, err := cmdExiv.CombinedOutput(); err != nil {
			log.Printf("exiv2 failed for job %s: %v\nOutput: %s", jobID, err, string(output))
			db.Model(&wallpaper).Update("Status", "failed")
			continue
		}

		args := []string{}
		if filepath.Ext(wallpaper.LightImg) == ".png" {
			args = append(args, "-L")
		}
		args = append(args, wallpaper.LightImg, wallpaper.DarkImg, "-o", finalPath)
		cmdHeif := exec.Command("heif-enc", args...)
		if output, err := cmdHeif.CombinedOutput(); err != nil {
			log.Printf("heif-enc failed for job %s: %v\nOutput: %s", jobID, err, string(output))
			db.Model(&wallpaper).Update("Status", "failed")
			continue
		}

		//  TODO: Upload `finalPath` to Cloudflare R2
		log.Printf("Uploading %s to R2...", finalPath)

		file, err := os.Open(finalPath)
		if err != nil {
			log.Printf("Failed to open final file for job %s: %v", jobID, err)
			db.Model(&wallpaper).Update("Status", "failed")
			continue
		}
		defer file.Close() // Make sure to close the file

		bucketName := os.Getenv("R2_BUCKET_NAME")
		if bucketName == "" {
			log.Printf("R2_BUCKET_NAME not set for job %s", jobID)
			db.Model(&wallpaper).Update("Status", "failed")
			continue
		}
		objectKey := "wallpapers/" + jobID + ".heic"

		_, err = s3Client.PutObject(ctx, &s3.PutObjectInput{
			Bucket:      aws.String(bucketName),
			Key:         aws.String(objectKey),
			Body:        file,
			ContentType: aws.String("image/heic"),
		})

		if err != nil {
			log.Printf("Failed to upload to R2 for job %s: %v", jobID, err)
			db.Model(&wallpaper).Update("Status", "failed")
			continue
		}

		publicURL := os.Getenv("R2_PUBLIC_URL")
		if publicURL == "" {
			log.Printf("R2_PUBLIC_URL not set for job %s", jobID)
			db.Model(&wallpaper).Update("Status", "failed")
			continue
		}
		finalURL := publicURL + "/" + objectKey
		log.Printf("Successfully uploaded. URL: %s", finalURL)

		updates := map[string]interface{}{"Status": "completed", "FinalURL": finalURL}
		if err := db.Model(&wallpaper).Updates(updates).Error; err != nil {
			log.Printf("Failed to update job %s to completed status: %v", jobID, err)
			continue
		}

		if err := os.RemoveAll(tmpDir); err != nil {
			log.Printf("Warning: failed to cleanup temp dir %s: %v", tmpDir, err)
		}

		log.Printf("Finished job %s", jobID)
	}
}
