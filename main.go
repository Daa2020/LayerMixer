package main

import (
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Layer struct {
	Name  string
	Image image.Image
}

type LayerCache map[string]image.Image

func readRandomLayersFromDirs(dirs []string) ([]Layer, error) {
	var layers []Layer

	for _, dir := range dirs {
		files, err := ioutil.ReadDir(dir)
		if err != nil {
			return nil, err
		}

		// Generate a random index within the range of the files slice
		rand.Seed(time.Now().UnixNano())
		randomIndex := rand.Intn(len(files))

		file := files[randomIndex]

		if !file.IsDir() {
			f, err := os.Open(filepath.Join(dir, file.Name()))
			if err != nil {
				return nil, err
			}

			img, err := png.Decode(f)
			if err != nil {
				return nil, err
			}

			layers = append(layers, Layer{Name: file.Name(), Image: img})

			err = f.Close()
			if err != nil {
				return nil, err
			}
		}
	}

	return layers, nil
}

func combineLayers(layers []Layer) image.Image {
	bounds := layers[0].Image.Bounds()
	combined := image.NewRGBA(bounds)

	draw.Draw(combined, bounds, layers[0].Image, image.Point{}, draw.Src)

	for _, layer := range layers[1:] {
		draw.Draw(combined, bounds, layer.Image, image.Point{}, draw.Over)
	}

	return combined
}

func getDirNames() []string {
	dirs := []string{}
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "DIR") {
			pair := strings.SplitN(env, "=", 2)
			dirs = append(dirs, os.Getenv(pair[0]))
		}
	}
	return dirs
}

func getNFTCount() int {
	nftCountStr := os.Getenv("NFT_COUNT")
	nftCount, err := strconv.Atoi(nftCountStr)
	if err != nil {
		log.Fatal(err) //"Invalid NFT_COUNT value")
	}
	return nftCount
}

func getOutputDir() string {
	outputDir := os.Getenv("OUTPUT_DIR")
	if outputDir == "" {
		log.Fatal("OUTPUT_DIR environment variable not set")
	}
	return outputDir
}

func createOutputDir(outputDir string) {
	if _, err := os.Stat(outputDir); !os.IsNotExist(err) {
		log.Fatalf("Output directory '%s' already exists", outputDir)
	}

	err := os.MkdirAll(outputDir, 0755)
	if err != nil {
		log.Fatal(err)
	}
}

func getCacheKey(layers []Layer) string {
	layerNames := make([]string, len(layers))
	for i, layer := range layers {
		layerNames[i] = layer.Name
	}
	cacheKey := strings.Join(layerNames, "-")
	return cacheKey
}

func getFromCache(cache LayerCache, layers []Layer) (image.Image, bool) {
	combined, ok := cache[getCacheKey(layers)]
	return combined, ok
}

func saveImageToFile(i int, img image.Image, outputDir string) {
	outFileName := fmt.Sprintf("%d.png", i)
	outFile, err := os.Create(filepath.Join(outputDir, outFileName))
	if err != nil {
		log.Fatal(err)
	}

	/* 	err = jpeg.Encode(outFile, img, &jpeg.Options{Quality: 90})
	   	if err != nil {
	   		log.Fatal(err)
	   	} */
	err = png.Encode(outFile, img)
	if err != nil {
		log.Fatal(err)
	}

}

func handlePanic() {
	if r := recover(); r != nil {
		fmt.Println("Program aborted due to a runtime error.")
		fmt.Println("Error:", r)
		//fmt.Println("Line of interruption:", debug.Stack())
	}
}

func main() {
	// Handle panics
	defer handlePanic()

	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	dirs := getDirNames()
	nftCount := getNFTCount()
	outputDir := getOutputDir()

	// Create the output directory
	createOutputDir(outputDir)

	// Create a cache to store combined layers
	cache := make(LayerCache)

	done := make(chan bool)

	// Loop through each NFT and generate a unique image for it
	for i := 1; i < nftCount+1; i++ {

		// Read a random set of layers from the specified directories
		layers, err := readRandomLayersFromDirs(dirs)
		if err != nil {
			log.Fatal("Error reading layers from dirs")
		}

		// Check if the combination of layers already exists in the cache
		combined, ok := getFromCache(cache, layers)
		if !ok {
			// If the combination of layers isn't in the cache, combine the layers to generate a unique image
			combined = combineLayers(layers)
			cache[getCacheKey(layers)] = combined
		} else {
			// If the combination of layers is in the cache, skip this iteration and move on to the next one
			fmt.Println(getCacheKey(layers), "already exists")
			continue
		}

		// Save the generated image to a file
		// saveImageToFile(i, combined, outputDir)
		go func(i int, combined image.Image, outputDir string) {
			saveImageToFile(i, combined, outputDir)
			done <- true
		}(i, combined, outputDir)

	}

	// Wait for all goroutines to finish executing
	for i := 1; i < nftCount+1; i++ {
		<-done
	}
}
