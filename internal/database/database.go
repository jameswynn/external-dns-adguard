package database

import (
	"fmt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"time"
)

type DnsEntry struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	EntryType string `gorm:"uniqueIndex:uidx_entry"`
	Namespace string `gorm:"uniqueIndex:uidx_entry"`
	Name      string `gorm:"uniqueIndex:uidx_entry"`
	Hostname  string `gorm:"uniqueIndex:uidx_hostname"`
	IP        string
}

func (entry DnsEntry) String() string {
	return fmt.Sprintf("%s/%s/%s='%s'='%s'", entry.EntryType, entry.Namespace, entry.Name, entry.Hostname, entry.IP)
}

type DnsDatabase struct {
	db *gorm.DB
}

func NewDnsDatabase(filename string) *DnsDatabase {
	db, err := gorm.Open(sqlite.Open(filename))
	if err != nil {
		panic("failed to connect database")
	}
	// Migrate the schema
	err = db.AutoMigrate(&DnsEntry{})
	if err != nil {
		panic("failed to migrate database")
	}

	return &DnsDatabase{
		db: db,
	}
}

func (dnsDb *DnsDatabase) GetByName(output *DnsEntry, entryType string, namespace string, name string) bool {
	result := dnsDb.db.
		Limit(1).
		Where(&DnsEntry{EntryType: entryType, Namespace: namespace, Name: name}).
		Find(&output)
	return result.RowsAffected > 0
}

func (dnsDb *DnsDatabase) GetByHostname(hostname string) (*DnsEntry, error) {
	var entry DnsEntry
	result := dnsDb.db.First(&entry, "hostname = ?", hostname)
	return &entry, result.Error
}

func (dnsDb *DnsDatabase) AddEntry(entry *DnsEntry) {
	dnsDb.db.Create(entry)
}

func (dnsDb *DnsDatabase) UpdateEntry(entry *DnsEntry) {
	dnsDb.db.Save(entry)
}

func (dnsDb *DnsDatabase) DeleteEntry(entry *DnsEntry) {
	dnsDb.db.Delete(&DnsEntry{}, entry.ID)
}

func (dnsDb *DnsDatabase) GetAll(entries *[]DnsEntry) {
	dnsDb.db.Limit(200).Find(entries)
}
