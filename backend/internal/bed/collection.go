package bed

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type CollectedBed struct {
	BedCode      string `json:"bedCode"`
	BedName      string `json:"bedName"`
	RoomCode     string `json:"roomCode"`
	BuildingCode string `json:"buildingCode"`
	Priority     int    `json:"priority"`
	BedCodes     string `json:"bedCodes"`
}

type Collection struct {
	Beds             []CollectedBed `json:"beds"`
	TotalConcurrency int            `json:"totalConcurrency"`
}

var (
	collection      Collection
	colMu           sync.RWMutex
	lastStudentCode string
)

func configDir() string {
	dir := os.Getenv("APPDATA")
	if dir == "" {
		dir = filepath.Join(os.Getenv("HOME"), ".config")
	}
	return filepath.Join(dir, "xjtu-housing-genius")
}

func collectionPath(studentCode string) string {
	return filepath.Join(configDir(), fmt.Sprintf("housing-config-%s.json", studentCode))
}

func LoadCollection(studentCode string) error {
	colMu.Lock()
	defer colMu.Unlock()
	lastStudentCode = studentCode
	collection = Collection{Beds: []CollectedBed{}, TotalConcurrency: 10}
	data, err := os.ReadFile(collectionPath(studentCode))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return json.Unmarshal(data, &collection)
}

func GetCollection() Collection {
	colMu.RLock()
	c := collection
	colMu.RUnlock()
	if len(c.Beds) == 0 && lastStudentCode != "" {
		LoadCollection(lastStudentCode)
		colMu.RLock()
		c = collection
		colMu.RUnlock()
	}
	return c
}

func SaveCollection(c Collection, studentCode string) error {
	colMu.Lock()
	collection = c
	lastStudentCode = studentCode
	colMu.Unlock()
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(collectionPath(studentCode), data, 0644)
}

func ValidatePriority(p int) error {
	if p < 1 || p > 5 {
		return fmt.Errorf("优先级必须在1-5之间")
	}
	return nil
}
