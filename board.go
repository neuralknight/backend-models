package models

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
)

// GameJoinMessage board
type GameJoinMessage struct {
	AgentURL url.URL
	ID       uuid.UUID
}

func openDB() *gorm.DB {
	db, err := gorm.Open("sqlite3", "chess.db")
	if err != nil {
		log.Panicln("failed to connect database", err, connStr)
	}

	tx := db.Begin()

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	// TODO(asakatida)
	// tx.DB().SetMaxIdleConns(10)
	// tx.DB().SetMaxOpenConns(100)
	// tx.DB().SetConnMaxLifetime(time.Hour)

	if !tx.HasTable(&boardModel{}) {
		tx.CreateTable(&boardModel{})
	}
	tx.AutoMigrate(&boardModel{})

	if !tx.HasTable(&agentModel{}) {
		tx.CreateTable(&agentModel{})
	}
	tx.AutoMigrate(&agentModel{})

	commitDB(tx)

	if errors := db.GetErrors(); len(errors) != 0 {
		panic(errors)
	}

	if errors := tx.GetErrors(); len(errors) != 0 {
		panic(errors)
	}

	return db.Begin()
}

func commitDB(db *gorm.DB) {
	errors := db.GetErrors()
	if len(errors) != 0 {
		panic(errors)
	}
	db.Commit()
}

func closeDB(db *gorm.DB) {
	defer func() {
		defer db.Close()
		if r := recover(); r != nil {
			db.Rollback()
			panic(r)
		}
	}()
	commitDB(db)
}

// Board board.
type Board interface {
	AddPlayer(decoder *json.Decoder) BoardStateMessage
	GetInfo(decoder *json.Decoder) BoardStateMessage
	GetState(values url.Values) BoardStateMessage
	GetStates(decoder *json.Decoder) BoardStateMessage
	PlayRound(decoder *json.Decoder) BoardStateMessage
}

// BoardInfoMessage board.
type BoardInfoMessage struct {
	Print string
}

// BoardCreatedMessage board.
type BoardCreatedMessage struct {
	ID uuid.UUID
}

// MakeGame agent.
func MakeGame(decoder *json.Decoder) BoardCreatedMessage {
	db := openDB()
	defer closeDB(db)
	var game boardModel
	game.ID = uuid.NewV5(uuid.NamespaceOID, "chess.board")
	game.State = initialBoard
	db.Create(&game)
	return BoardCreatedMessage{game.ID}
}

// GetGame game.
func GetGame(ID uuid.UUID) Board {
	db := openDB()
	defer closeDB(db)
	var game boardModel
	rows, err := db.First(&game, "id = ?", ID).Rows()
	if err != nil {
		log.Panicln(err)
	}
	if !rows.Next() {
		log.Panicln(rows)
	}
	err = rows.Scan(&game.ID, &game.CreatedAt, &game.UpdatedAt, &game.DeletedAt, &game.State, &game.MoveCount, &game.MovesSincePawn, &game.Player1, &game.Player2)
	if err != nil {
		log.Panicln(err)
	}
	if game.ID != ID {
		panic(game)
	}
	if game.ID.Version() != uuid.V5 {
		panic(game)
	}
	return &game
}

// GetGames game.
func GetGames(decoder *json.Decoder) BoardStatesMessage {
	db := openDB()
	defer closeDB(db)
	var game boardModel
	rows, err := db.Find(&game).Rows()
	if err != nil {
		log.Panicln("Failed to get game rows", err)
	}
	defer rows.Close()
	games := make([]uuid.UUID, 0)
	for rows.Next() {
		var game boardModel
		err := rows.Scan(&game.ID, &game.CreatedAt, &game.UpdatedAt, &game.DeletedAt, &game.State, &game.MoveCount, &game.MovesSincePawn, &game.Player1, &game.Player2)
		if err != nil {
			log.Panicln("Failed to scan row", err)
		}
		games = append(games, game.ID)
	}
	if err := rows.Err(); err != nil {
		log.Panicln("Error iterating rows", err)
	}
	return BoardStatesMessage{games}
}

// AddPlayer to board.
func (board boardModel) stateMessage() BoardStateMessage {
	return BoardStateMessage{End: !board.active(), State: board.State}
}

// AddPlayer to board.
func (board *boardModel) AddPlayer(decoder *json.Decoder) BoardStateMessage {
	if board.Player2.Version() == uuid.V5 {
		log.Panicln("Game is full.")
	}
	var message GameJoinMessage
	err := decoder.Decode(&message)
	if err != nil {
		log.Panicln(err)
	}
	if message.ID.Version() != uuid.V5 {
		log.Panicln("Invalid agent ID")
	}
	if board.Player1.Version() != uuid.V5 {
		board.Player1 = message.ID
	} else {
		board.Player2 = message.ID
		// Player 2 joins game.
		board.pokePlayer(board.Player1)
	}
	return board.stateMessage()
}

// Inform active player of game state.
func (board boardModel) pokePlayer(player uuid.UUID) {
	data, err := json.Marshal(board.stateMessage())
	if err != nil {
		log.Panicln(err)
	}
	req, err := http.NewRequest(http.MethodPut, "", bytes.NewReader(data))
	if err != nil {
		log.Panicln(err)
	}
	defer req.Body.Close()
	var client http.Client
	resp, err := client.Do(req)
	if err != nil {
		log.Panicln(err)
	}
	defer resp.Body.Close()
	var message BoardStateMessage
	err = json.NewDecoder(resp.Body).Decode(&message)
	if err != nil {
		log.Panicln(err)
	}
	// self.request("PUT", f"/agent/{ active_player or self.active_player() }", json={"end": end})
}

// GetInfo game.
func (board boardModel) GetInfo(decoder *json.Decoder) BoardStateMessage {
	return BoardStateMessage{}
}

// GetState game.
func (board boardModel) GetState(values url.Values) BoardStateMessage {
	return BoardStateMessage{!board.active(), false, board.State}
}

// GetStates game.
func (board boardModel) GetStates(decoder *json.Decoder) BoardStateMessage {
	return BoardStateMessage{}
}

// PlayRound game.
func (board boardModel) PlayRound(decoder *json.Decoder) BoardStateMessage {
	return BoardStateMessage{}
}

// class BlankBoard:
//     def __str__(self):
//         return "\n" * 8
//
//     def close(self, *args, **kwargs):
//         return {"end": True}
//
//     def current_state_v1(self, *args, **kwargs):
//         return {"state": {"end": True}}
//
//     def slice_cursor_v1(self, *args, **kwargs):
//         return {"cursor": None, "boards": []}
//
//     def update_state_v1(self, *args, **kwargs):
//         return {"end": True}
//
//
// def get_game(request):
//     """
//     Retrieve board for request.
//     """
//     try:
//         return Board.get_game(request.matchdict["game"])
//     except NoBoard:
//         pass
//     return BlankBoard()
//
//
// @games.get()
// def get_games(request):
//     """
//     Retrieve all game ids.
//     """
//     return {"ids": tuple(Board.GAMES.keys())}
//
//
// @games.post()
// def post_game(request):
//     """
//     Create a new game and provide an id for interacting.
//     """
//     return {"id": Board().id}
//
//
// @game_states.get()
// def get_states(request):
//     """
//     Start or continue a cursor of next board states.
//     """
//     cursor = get_game(request).slice_cursor_v1(**request.GET)
//     cursor["boards"] = tuple(map(
//         lambda boards: tuple(map(
//             lambda board: tuple(map(methodcaller("hex"), board)),
//             boards)),
//         cursor["boards"]))
//     return cursor
//
//
// @game_interaction.get()
// def get_state(request):
//     """
//     Provide current state on the board.
//     """
//     state = get_game(request).current_state_v1()
//     if isinstance(state["state"], dict):
//         return state
//     state["state"] = tuple(map(methodcaller("hex"), state["state"]))
//     return state
//
//
// @game_interaction.put()
// def put_state(request):
//     """
//     Make a move to a new state on the board.
//     """
//     try:
//         return get_game(request).update_state_v1(request.dbsession, **request.json)
//     except InvalidMove:
//         pass
//     return {"invalid": True}
//
//
// @game_info.get()
// def get_info(request):
//     """
//     Provide current state on the board.
//     """
//     return {"print": str(get_game(request))}
// """
// Chess state handling model.
// """
//
// from concurrent.futures import ThreadPoolExecutor
// from itertools import count
// from json import dumps
//
// from .base_board import BaseBoard, NoBoard
// from .table_board import TableBoard
// from .table_game import TableGame
//
// __all__ = ["Board", "NoBoard"]
//
//
// class Board(BaseBoard):
//     """
//     Chess board interaction model.
//     """
//
//     EMOJI = [
//       "⌛", "‼",
//       "♝", "♗", "♚", "♔", "♞", "♘", "♟", "♙", "♛", "♕", "♜", "♖", "▪", "▫"]
//
//     def __init__(self, board=None, _id=None, active_player=True):
//         """
//         Set up board.
//         """
//         super().__init__(board, _id, active_player)
//         self.executor = ThreadPoolExecutor()
//
//     def __repr__(self):
//         """
//         Output the raw view of board.
//         """
//         return f"Board({ self.board !r})"
//
//     def __str__(self):
//         """
//         Output the emoji view of board.
//         """
//         if self._active_player:
//             def piece_to_index(piece):
//                 return (piece & 0xF)
//         else:
//             def piece_to_index(piece):
//                 return (piece & 0xE) | (0 if piece & 1 else 1)
//
//         return "\n".join(map(
//             lambda posY, row: "".join(map(
//                 lambda posX, piece: self.EMOJI[
//                     piece_to_index(piece)
//                     if piece else
//                     14 + ((posY + posX) % 2)],
//                 count(), row)),
//             count(),
//             self.board if self._active_player else reversed(
//                 [reversed(row) for row in self.board])))
//
//     def slice_cursor_v1(self, cursor=None, lookahead=1, complete=False):
//         """
//         Retrieve REST cursor slice.
//         """
//         return self.cursor_delegate.slice_cursor_v1(self._board, cursor, int(lookahead), complete)
//
//     def update_state_v1(self, dbsession, state):
//         """
//         Make a move to a new state on the board.
//         """
//         moving_player = self.active_player()
//         board = self.update(state)
//         table_game = dbsession.query(TableGame).filter(
//             TableGame.game == board.id).first()
//         table_board = TableBoard(
//             board_state=dumps(tuple(map(tuple, board.board))),
//             move_num=board._board.move_count,
//             player=board.active_player(),
//             game=board.id)
//         if table_game:  # TODO(asakatida)
//             table_board.game_link.append(table_game)
//         dbsession.add(table_board)
//         if board:
//             board.poke_player(False)
//             return {"end": False}
//         board.poke_player(True, moving_player)
//         if board._board.has_kings():
//             table_game.one_won = False
//             table_game.two_won = False
//         elif moving_player == table_game.player_one:
//             table_game.two_won = False
//         else:
//             table_game.one_won = False
//         board.close()
//         return {"end": True}
// """
// Chess state.
//
// Dumps all board state mappings into rethinkdb.
// """
//
// from asyncio import Queue
// from asyncio import TimeoutError as AsyncTimeoutError
// from asyncio import get_event_loop, sleep, wait
// from asyncio.queues import QueueEmpty
// from functools import partial
// from hashlib import sha256
// from itertools import chain, count, filterfalse, repeat
// from sys import argv
//
// import rethinkdb as r
//
// from .const import (BISHOP, BISHOP_MOVES, KING, KING_MOVES, KNIGHT,
//                     KNIGHT_MOVES, QUEEN, QUEEN_MOVES, ROOK, ROOK_MOVES,
//                     _to_emoji)
//
//
// def _active_piece(piece):
//     return piece & 1 and piece & 0xE
//
//
// def _active_pieces(board):
//     def _row(posY, row):
//         def _piece(posX, piece):
//             nonlocal posY
//             if _active_piece(piece):
//                 return piece, posX, posY
//             return None
//         return map(_piece, count(), row)
//     return filter(None, chain.from_iterable(map(_row, count(), board)))
//
//
// def _inactive_piece(piece):
//     return (not piece & 1) and piece & 0xE
//
//
// def _is_on_board(posX, posY, move):
//     return 0 <= (posX + move[0]) < 8 and 0 <= (posY + move[1]) < 8
//
//
// def _sha256(b):
//     m = sha256()
//     m.update(b)
//     return m.digest()
//
//
// def _lookahead_boards_for_board(board):
//     raw = board["raw"]
//     return map(
//         partial(
//             _lookahead_boards_to_table,
//             board["id"],
//             board["move"]),
//         chain.from_iterable(
//             map(
//                 _lookahead_boards_for_piece,
//                 repeat(raw),
//                 repeat(_lookahead_check(raw)),
//                 _active_pieces(raw))))
//
//
// def _lookahead_boards_to_table(_id, move, next_board):
//     return (
//         {
//             "id": b"".join(next_board),
//             "emoji": _to_emoji(next_board),
//             "move": move + 1,
//             "raw": next_board,
//             "status": False},
//         {
//             "id": _sha256(b" ".join((_id, b"".join(next_board)))),
//             "parent": _id,
//             "child": b"".join(next_board)})
//
//
// def _lookahead_boards_for_piece(board, check, piece):
//     piece, posX, posY = piece
//     valid_moves_for_piece = _valid_moves_for_piece(board, piece, posX, posY)
//     if check:
//         valid_moves_for_piece = filter(
//             lambda move: board[posY + move[1]][posX + move[0]] & 0xE == KING,
//             valid_moves_for_piece)
//         return chain.from_iterable(map(
//             lambda move: _mutate_board(board, move, piece & 0xF, posX, posY),
//             valid_moves_for_piece))
//     return filterfalse(
//         _lookahead_check,
//         chain.from_iterable(map(
//             lambda move: _mutate_board(board, move, piece & 0xF, posX, posY),
//             valid_moves_for_piece)))
//
//
// def _lookahead_check(board):
//     return any(chain.from_iterable(map(
//         _lookahead_check_for_piece,
//         repeat(board),
//         _active_pieces(board))))
//
//
// def _lookahead_check_for_piece(board, piece):
//     piece, posX, posY = piece
//     return map(
//         lambda move: (board[posY + move[1]][posX + move[0]] & 0xF) == KING,
//         _valid_moves_for_piece(board, piece, posX, posY))
//
//
// def _moves_for_pawn(board, piece, posX, posY):
//     if (
//             _is_on_board(posX, posY, (0, -1))
//             and (not board[posY - 1][posX])):
//         yield (0, -1)
//     if (
//             posY == 6
//             and (not board[posY - 1][posX])
//             and (not board[posY - 2][posX])):
//         yield (0, -2)
//     if (
//             _is_on_board(posX, posY, (-1, -1))
//             and _inactive_piece(board[posY - 1][posX - 1])):
//         yield (-1, -1)
//     if (
//             _is_on_board(posX, posY, (1, -1))
//             and _inactive_piece(board[posY - 1][posX + 1])):
//         yield (1, -1)
//
//
// def _moves_for_piece(board, piece, posX, posY):
//     return filter(
//         partial(_is_on_board, posX, posY),
//         (
//             (),  # No piece
//             BISHOP_MOVES,
//             KING_MOVES,
//             KNIGHT_MOVES,
//             _moves_for_pawn(board, piece, posX, posY),
//             QUEEN_MOVES,
//             ROOK_MOVES
//         )[(piece & 0xE) // 2])
//
//
// def _mutate_board(board, move, piece, posX, posY):
//     new_state = list(map(list, board))
//     new_state[posY][posX] = 0
//     if piece == 9 and posY == 1:
//         for promote in (BISHOP, KNIGHT, QUEEN, ROOK):
//             new_state[posY + move[1]][posX + move[0]] = promote | 1
//             yield _swap(new_state)
//     new_state[posY + move[1]][posX + move[0]] = piece
//     yield _swap(new_state)
//
//
// def _swap(board):
//     return tuple(map(
//         lambda row: bytes(map(
//             lambda pp: (pp ^ 1) if pp else 0,
//             row))[::-1],
//         board))[::-1]
//
//
// def _unit(i):
//     return -1 if i < 0 else (0 if i == 0 else 1)
//
//
// def _valid_moves_for_piece(board, piece, posX, posY):
//     return filter(
//         _validation_for_piece(board, piece, posX, posY),
//         _moves_for_piece(board, piece, posX, posY))
//
//
// def _validate_ending(board, posX, posY, move):
//     return not _active_piece(
//         board[posY + move[1]][posX + move[0]])
//
//
// def _validate_move(board, posX, posY, move):
//     return (
//         _validate_ending(board, posX, posY, move)
//         and all(
//             map(
//                 lambda _range:
//                 not board
//                 [posY + _unit(move[1]) * _range]
//                 [posX + _unit(move[0]) * _range],
//                 range(1, max(abs(move[0]), abs(move[1]))))))
//
//
// def _validation_for_piece(board, piece, posX, posY):
//     return partial((
//         lambda *args: False,  # No piece
//         partial(_validate_move, board),  # Bishop
//         partial(_validate_ending, board),  # King
//         partial(_validate_ending, board),  # Knight
//         lambda *args: True,  # Pawn
//         partial(_validate_move, board),  # Queen
//         partial(_validate_move, board)  # Rook
//         )[(piece & 0xE) // 2], posX, posY)
//
//
// async def _consumers(queue):
//     loop = get_event_loop()
//     await wait([
//         await loop.run_in_executor(None, _consumer, queue)
//         for _ in range(5)])
//
//
// def _get_task_done(queue):
//     value = queue.get_nowait()
//     queue.task_done()
//     return value
//
//
// async def _get_task(queue):
//     try:
//         return _get_task_done(queue)
//     except QueueEmpty:
//         await sleep(60)
//     return _get_task_done(queue)
//
//
// async def _consumer(queue):
//     conn = await r.connect(db="chess")
//     try:
//         while True:
//             try:
//                 boards = list(
//                     _lookahead_boards_for_board(await _get_task(queue)))
//             except QueueEmpty:
//                 break
//             try:
//                 while len(boards) < 1000:
//                     boards.extend(
//                         _lookahead_boards_for_board(await _get_task(queue)))
//             except QueueEmpty:
//                 pass
//             boards, links = zip(*boards)
//             await r.table("boards").insert(
//                 boards,
//                 conflict=lambda id, old, new: r.branch(
//                     old["move"] < new["move"], old, new),
//                 durability="soft"
//             ).run(conn, noreply=True)
//             await r.table("links").insert(links).run(conn)
//             await r.table("boards").get_all(
//                 r.args(set(link["parent"] for link in links))
//             ).update({"status": True}).run(conn, noreply=True)
//             if len(boards) < 1000:
//                 break
//     finally:
//         conn.close()
//
//
// async def _async_main():
//     loop = get_event_loop()
//     queue = Queue(997)
//     loop.create_task(_consumers(queue))
//     conn = await r.connect(db="chess")
//     try:
//         try:
//             (
//                 r.table("boards")
//                 .index_create(
//                     "status",
//                     lambda doc: doc["status"].default(False))
//                 .run(conn))
//         except r.errors.ReqlOpFailedError:
//             pass
//         r.table("boards").index_wait().run(conn)
//         validate = await (
//             r.table("boards")
//             .get_all(False, index="status")
//             .filter(lambda row: row["move"] < int(argv[1]))
//             .limit(1)
//             .run(conn))
//         if not await validate.fetch_next():
//             return print("no work done")
//         feed = await (
//             r.table("boards")
//             .get_all(False, index="status")
//             .changes(include_initial=True, squash=2)
//             .filter({"old_val": None}, default=True)
//             .with_fields("new_val")["new_val"]
//             .with_fields("id", "move", "raw")
//             .filter(lambda row: row["move"] < int(argv[1]))
//             .run(conn))
//         await feed.fetch_next()
//         print("generating")
//         try:
//             while await feed.fetch_next(60):
//                 await queue.put(await feed.next())
//         except AsyncTimeoutError:
//             pass
//         except r.errors.ReqlTimeoutError:
//             pass
//     finally:
//         conn.close()
//         print("finalizing")
//         await queue.join()
//
//
// def _main():
//     r.set_loop_type("asyncio")
//     loop = get_event_loop()
//     loop.run_until_complete(_async_main())
//
//
// if __name__ == "__main__":
//     try:
//         _main()
//     except KeyboardInterrupt:
//         print()
