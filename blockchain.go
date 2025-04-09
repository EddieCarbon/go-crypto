package main

import (
	"fmt"
	"log"

	"github.com/boltdb/bolt"
)

const dbFile = "blockchain.db"
const blocksBucket = "blocks"

// Blockchain keeps a sequence of Blocks
type Blockchain struct {
	tip []byte
	db  *bolt.DB
}

// BlockchainIterator is used to iterate over blockchain blocks
type BlockchainIterator struct {
	currentHash []byte
	db          *bolt.DB
}

// AddBlock saves provided data as a block in the blockchain
func (bc *Blockchain) AddBlock(data string) {
	var lastHash []byte

	if err := bc.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		lastHash = b.Get([]byte("l"))
		return nil
	}); err != nil {
		log.Panic(err)
	}

	newBlock := NewBlock(data, lastHash)

	if err := bc.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))

		if err := b.Put(newBlock.Hash, newBlock.Serialize()); err != nil {
			return err
		}

		if err := b.Put([]byte("l"), newBlock.Hash); err != nil {
			return err
		}

		bc.tip = newBlock.Hash

		return nil
	}); err != nil {
		log.Panic(err)
	}
}

func (bc *Blockchain) Iterator() *BlockchainIterator {
	bci := &BlockchainIterator{bc.tip, bc.db}

	return bci
}

func (i *BlockchainIterator) Next() *Block {
	var block *Block

	err := i.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		encodedBlock := b.Get(i.currentHash)
		block = DeserializeBlock(encodedBlock)

		return nil
	})

	if err != nil {
		log.Panic(err)
	}

	i.currentHash = block.PrevBlockHash

	return block
}

// NewBlockchain creates a new Blockchain with genesis Block
func NewBlockchain() *Blockchain {
	var tip []byte
	db, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		log.Panic(err)
	}

	if err := db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		if b == nil {
			return initializeNewBlockchain(tx, &tip)
		}

		tip = b.Get([]byte("l"))

		return nil
	}); err != nil {
		log.Panic(err)
	}

	bc := Blockchain{tip, db}

	return &bc
}

// InitializeNewBlockchain initializes a new blockchain with genesis block
func initializeNewBlockchain(tx *bolt.Tx, tip *[]byte) error {
	fmt.Println("No existing blockchain found. Creating a new one...")
	genesis := NewGenesisBlock()

	b, err := tx.CreateBucket([]byte(blocksBucket))
	if err != nil {
		return err
	}

	if err = b.Put(genesis.Hash, genesis.Serialize()); err != nil {
		return err
	}

	if err = b.Put([]byte("l"), genesis.Hash); err != nil {
		return err
	}

	*tip = genesis.Hash
	return nil
}
