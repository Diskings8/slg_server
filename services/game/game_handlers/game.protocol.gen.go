package game_handlers

// 协议注册 — 手工维护，后续可由 game_generates 自动生成
//
// 注册方式：在 init() 中调用 RegisterProto()
// 新协议三步：
//  1. protocol.proto 加 MsgID
//  2. handler 包加处理函数
//  3. 此文件追加 RegisterProto()
