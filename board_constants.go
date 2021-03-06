package models

import (
	"database/sql/driver"
	"encoding/hex"
	"encoding/json"
	"strings"
)

type board [8][8]uint8

func (b *board) from(state [8]string) error {
	for i, r := range state {
		row, err := hex.DecodeString(r)
		if err != nil {
			return err
		}
		if len(row) != 8 {
			return err
		}
		copy(b[i][:], row)
	}
	return nil
}

func (b board) to() [8]string {
	var state [8]string
	for i, r := range b {
		state[i] = hex.EncodeToString(r[:])
	}
	return state
}

func (b *board) UnmarshalJSON(bytes []byte) error {
	var state [8]string
	if err := json.Unmarshal(bytes, &state); err != nil {
		return err
	}
	return b.from(state)
}

func (b board) Value() (driver.Value, error) {
	state := b.to()
	return strings.Join(state[:], ","), nil
}

func (b board) Scan(cell interface{}) error {
	switch cell := cell.(type) {
	case string:
		var state [8]string
		copy(state[:], strings.Split(cell, ","))
		b.from(state)
	default:
		return nil
	}
	return nil
}

func (b board) MarshalJSON() ([]byte, error) {
	return json.Marshal(b.to())
}

// BISHOP piece.
const (
	BISHOP uint8 = 2
	KING   uint8 = 4
	KNIGHT uint8 = 6
	PAWN   uint8 = 8
	QUEEN  uint8 = 10
	ROOK   uint8 = 12
)

// low bit indicates active player piece
var initialBoard = board{
	{ROOK | 0x10, KNIGHT, BISHOP, QUEEN, KING | 0x10, BISHOP, KNIGHT, ROOK | 0x10},
	{PAWN, PAWN, PAWN, PAWN, PAWN, PAWN, PAWN, PAWN},
	{}, {}, {}, {},
	{PAWN | 1, PAWN | 1, PAWN | 1, PAWN | 1, PAWN | 1, PAWN | 1, PAWN | 1, PAWN | 1},
	{ROOK | 0x11, KNIGHT | 1, BISHOP | 1, QUEEN | 1, KING | 0x11, BISHOP | 1, KNIGHT | 1, ROOK | 0x11}}

var (
	bishopMoves = [32][2]int8{
		{-8, -8}, {-7, -7}, {-6, -6}, {-5, -5},
		{-4, -4}, {-3, -3}, {-2, -2}, {-1, -1},
		{1, 1}, {2, 2}, {3, 3}, {4, 4},
		{5, 5}, {6, 6}, {7, 7}, {8, 8},
		{-8, 8}, {-7, 7}, {-6, 6}, {-5, 5},
		{-4, 4}, {-3, 3}, {-2, 2}, {-1, 1},
		{1, -1}, {2, -2}, {3, -3}, {4, -4},
		{5, -5}, {6, -6}, {7, -7}, {8, -8},
	}
	kingMoves = [8][2]int8{
		{-1, -1}, {-1, 0}, {-1, 1}, {0, -1},
		{0, 1}, {1, -1}, {1, 0}, {1, 1},
	}
	knightMoves = [8][2]int8{
		{-2, -1}, {-2, 1}, {-1, -2}, {-1, 2},
		{1, -2}, {1, 2}, {2, -1}, {2, 1},
	}
	queenMoves = [64][2]int8{
		{-8, -8}, {-7, -7}, {-6, -6}, {-5, -5},
		{-4, -4}, {-3, -3}, {-2, -2}, {-1, -1},
		{1, 1}, {2, 2}, {3, 3}, {4, 4},
		{5, 5}, {6, 6}, {7, 7}, {8, 8},
		{-8, 8}, {-7, 7}, {-6, 6}, {-5, 5},
		{-4, 4}, {-3, 3}, {-2, 2}, {-1, 1},
		{1, -1}, {2, -2}, {3, -3}, {4, -4},
		{5, -5}, {6, -6}, {7, -7}, {8, -8},
		{0, -8}, {0, -7}, {0, -6}, {0, -5},
		{0, -4}, {0, -3}, {0, -2}, {0, -1},
		{0, 1}, {0, 2}, {0, 3}, {0, 4},
		{0, 5}, {0, 6}, {0, 7}, {0, 8},
		{-8, 0}, {-7, 0}, {-6, 0}, {-5, 0},
		{-4, 0}, {-3, 0}, {-2, 0}, {-1, 0},
		{1, 0}, {2, 0}, {3, 0}, {4, 0},
		{5, 0}, {6, 0}, {7, 0}, {8, 0},
	}
	rookMoves = [32][2]int8{
		{0, -8}, {0, -7}, {0, -6}, {0, -5},
		{0, -4}, {0, -3}, {0, -2}, {0, -1},
		{0, 1}, {0, 2}, {0, 3}, {0, 4},
		{0, 5}, {0, 6}, {0, 7}, {0, 8},
		{-8, 0}, {-7, 0}, {-6, 0}, {-5, 0},
		{-4, 0}, {-3, 0}, {-2, 0}, {-1, 0},
		{1, 0}, {2, 0}, {3, 0}, {4, 0},
		{5, 0}, {6, 0}, {7, 0}, {8, 0},
	}
)

func unit(i int8) int8 {
	if i < 0 {
		return -1
	}
	if i == 0 {
		return 0
	}
	return 1
}
