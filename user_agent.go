package neuralknightmodels

import (
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
)

// UserAgent Human Agent
type userAgent struct {
	simpleAgent
}

type userMoveMessage struct {
	move [2][2]int
}

func getMove(r io.Reader) [2][2]int {
	var message userMoveMessage
	err := json.NewDecoder(r).Decode(message)
	if err != nil {
		panic(err)
	}
	return message.move
}

// PlayRound Play a game round
func (agent userAgent) PlayRound(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	move := getMove(r.Body)
	proposal := agent.getState()
	if proposal.end {
		json.NewEncoder(w).Encode(proposal)
		return
	}
	var out board
	for i, r := range proposal.state {
		row, err := hex.DecodeString(r)
		if err != nil {
			panic(err)
		}
		if len(row) != 8 {
			panic(row)
		}
		copy(out[i][:], row)
	}
	out[move[1][0]][move[1][1]] = out[move[0][0]][move[0][1]]
	out[move[0][0]][move[0][1]] = 0
	resp := agent.putBoard(out)
	defer resp.Body.Close()
	var message stateMessage
	err := json.NewDecoder(resp.Body).Decode(message)
	if err != nil {
		panic(err)
	}
	agent.gameOver = message.end
	if agent.gameOver {
		agent.close(w, r)
		return
	}
	json.NewEncoder(w).Encode(message)
}
