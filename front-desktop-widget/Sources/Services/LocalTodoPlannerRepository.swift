import Foundation
import SQLite3

/// SQLite-backed repository that caches data locally and syncs with remote API
/// Uses native SQLite3 C library (no external dependencies)
final class LocalTodoPlannerRepository: @unchecked Sendable, TodoPlannerRepository {
  private var db: OpaquePointer?
  private let remoteAPI = APIClient.shared
  private let dbPath: String

  init(dbPath: String) throws {
    self.dbPath = dbPath
    try openDatabase()
    try createSchema()
    print("[SQLite] Initialized at \(dbPath)")
  }

  deinit {
    sqlite3_close(db)
  }

  // MARK: - Database Utilities

  private func openDatabase() throws {
    let flags = SQLITE_OPEN_CREATE | SQLITE_OPEN_READWRITE | SQLITE_OPEN_FULLMUTEX
    if sqlite3_open_v2(dbPath, &db, flags, nil) != SQLITE_OK {
      throw StorageError.databaseError("Failed to open database at \(dbPath)")
    }
  }

  private func createSchema() throws {
    let schema = """
      CREATE TABLE IF NOT EXISTS categories (
        id INTEGER PRIMARY KEY,
        name TEXT NOT NULL,
        color TEXT NOT NULL,
        pomodoro_work_duration INTEGER,
        pomodoro_rest_duration INTEGER,
        created_at TEXT NOT NULL,
        updated_at TEXT NOT NULL
      );

      CREATE TABLE IF NOT EXISTS day_records (
        id INTEGER PRIMARY KEY,
        calendar_date TEXT NOT NULL UNIQUE,
        review_status TEXT NOT NULL,
        created_at TEXT NOT NULL,
        updated_at TEXT NOT NULL
      );

      CREATE TABLE IF NOT EXISTS actual_blocks (
        id INTEGER PRIMARY KEY,
        day_record_id INTEGER NOT NULL,
        category_id INTEGER,
        block_type TEXT NOT NULL,
        start_time TEXT NOT NULL,
        duration_minutes INTEGER NOT NULL,
        FOREIGN KEY (day_record_id) REFERENCES day_records(id) ON DELETE CASCADE
      );

      CREATE INDEX IF NOT EXISTS idx_actual_blocks_day_record ON actual_blocks(day_record_id);

      CREATE TABLE IF NOT EXISTS pending_events (
        local_id INTEGER PRIMARY KEY AUTOINCREMENT,
        day_record_id INTEGER NOT NULL,
        event_type TEXT NOT NULL,
        outgoing_category_id INTEGER,
        incoming_category_id INTEGER,
        occurred_at TEXT NOT NULL,
        synced INTEGER NOT NULL DEFAULT 0,
        FOREIGN KEY (day_record_id) REFERENCES day_records(id) ON DELETE CASCADE
      );

      CREATE INDEX IF NOT EXISTS idx_pending_events_day_record ON pending_events(day_record_id);
      CREATE INDEX IF NOT EXISTS idx_pending_events_synced ON pending_events(synced);
      """

    var error: UnsafeMutablePointer<CChar>?
    if sqlite3_exec(db, schema, nil, nil, &error) != SQLITE_OK {
      let errorMessage = String(cString: error!)
      sqlite3_free(error)
      throw StorageError.databaseError("Failed to create schema: \(errorMessage)")
    }
  }

  private func execute(_ sql: String, params: [Any] = []) throws {
    var statement: OpaquePointer?
    defer { sqlite3_finalize(statement) }

    if sqlite3_prepare_v2(db, sql, -1, &statement, nil) != SQLITE_OK {
      throw StorageError.databaseError("Failed to prepare: \(sql)")
    }

    try bindParams(statement!, params: params)

    if sqlite3_step(statement) != SQLITE_DONE {
      let errorMessage = String(cString: sqlite3_errmsg(db))
      throw StorageError.databaseError("Failed to execute: \(errorMessage)")
    }
  }

  private func query<T>(_ sql: String, params: [Any] = [], mapper: (OpaquePointer) throws -> T)
    throws
    -> [T]
  {
    var statement: OpaquePointer?
    defer { sqlite3_finalize(statement) }

    if sqlite3_prepare_v2(db, sql, -1, &statement, nil) != SQLITE_OK {
      throw StorageError.databaseError("Failed to prepare query: \(sql)")
    }

    try bindParams(statement!, params: params)

    var results: [T] = []
    while sqlite3_step(statement) == SQLITE_ROW {
      results.append(try mapper(statement!))
    }

    return results
  }

  private func queryOne<T>(
    _ sql: String, params: [Any] = [], mapper: (OpaquePointer) throws -> T
  ) throws -> T? {
    let results: [T] = try query(sql, params: params, mapper: mapper)
    return results.first
  }

  private func bindParams(_ statement: OpaquePointer, params: [Any]) throws {
    for (index, param) in params.enumerated() {
      let idx = Int32(index + 1)

      switch param {
      case let value as Int:
        sqlite3_bind_int(statement, idx, Int32(value))
      case let value as Int?:
        if let v = value {
          sqlite3_bind_int(statement, idx, Int32(v))
        } else {
          sqlite3_bind_null(statement, idx)
        }
      case let value as String:
        sqlite3_bind_text(statement, idx, (value as NSString).utf8String, -1, nil)
      case let value as String?:
        if let v = value {
          sqlite3_bind_text(statement, idx, (v as NSString).utf8String, -1, nil)
        } else {
          sqlite3_bind_null(statement, idx)
        }
      case let value as Date:
        let iso = ISO8601DateFormatter().string(from: value)
        sqlite3_bind_text(statement, idx, (iso as NSString).utf8String, -1, nil)
      case let value as Bool:
        sqlite3_bind_int(statement, idx, value ? 1 : 0)
      default:
        throw StorageError.databaseError("Unsupported parameter type at index \(index)")
      }
    }
  }

  private func columnString(_ statement: OpaquePointer, index: Int32) -> String {
    return String(cString: sqlite3_column_text(statement, index))
  }

  private func columnInt(_ statement: OpaquePointer, index: Int32) -> Int {
    return Int(sqlite3_column_int(statement, index))
  }

  private func columnIntOptional(_ statement: OpaquePointer, index: Int32) -> Int? {
    if sqlite3_column_type(statement, index) == SQLITE_NULL {
      return nil
    }
    return Int(sqlite3_column_int(statement, index))
  }

  private func columnDate(_ statement: OpaquePointer, index: Int32) -> Date {
    let iso = columnString(statement, index: index)
    return ISO8601DateFormatter().date(from: iso) ?? Date()
  }

  private func columnBool(_ statement: OpaquePointer, index: Int32) -> Bool {
    return sqlite3_column_int(statement, index) != 0
  }

  // MARK: - Authentication

  func getAuthToken() -> String? {
    return UserDefaults.standard.string(forKey: "com.todoplanner.widget.jwt_token")
  }

  func persistAuthToken(_ token: String) async throws {
    remoteAPI.setAuthToken(token)
  }

  func clearAuth() async throws {
    remoteAPI.clearAuthToken()
    try clearLocalData()
  }

  func validateAuth() async throws -> Bool {
    return await remoteAPI.validateToken()
  }

  // MARK: - Categories

  func fetchCategories() async throws -> [Category] {
    // Try to fetch from remote first
    do {
      let remoteCategories = try await remoteAPI.fetchCategories()
      // Cache them locally
      try await cacheCategories(remoteCategories)
      return remoteCategories
    } catch {
      print("[SQLite] Remote fetch failed, falling back to cache: \(error)")
      // Fall back to cached data
      return try loadCategoriesFromCache()
    }
  }

  private func cacheCategories(_ categories: [Category]) async throws {
    // Clear and insert in a transaction
    try execute("BEGIN TRANSACTION")
    do {
      try execute("DELETE FROM categories")

      for category in categories {
        let sql = """
          INSERT INTO categories (id, name, color, pomodoro_work_duration, pomodoro_rest_duration, created_at, updated_at)
          VALUES (?, ?, ?, ?, ?, ?, ?)
          """
        let iso8601 = ISO8601DateFormatter()
        try execute(
          sql,
          params: [
            category.id,
            category.name,
            category.color,
            category.pomodoroConfig?.workDuration as Any,
            category.pomodoroConfig?.restDuration as Any,
            iso8601.string(from: category.createdAt),
            iso8601.string(from: category.updatedAt),
          ])
      }

      try execute("COMMIT")
    } catch {
      try execute("ROLLBACK")
      throw error
    }
  }

  private func loadCategoriesFromCache() throws -> [Category] {
    let sql = "SELECT * FROM categories ORDER BY id"
    return try query(sql) { stmt in
      let pomodoroConfig: PomodoroConfig?
      if let workDuration = columnIntOptional(stmt, index: 3),
        let restDuration = columnIntOptional(stmt, index: 4)
      {
        pomodoroConfig = PomodoroConfig(workDuration: workDuration, restDuration: restDuration)
      } else {
        pomodoroConfig = nil
      }

      return Category(
        id: columnInt(stmt, index: 0),
        name: columnString(stmt, index: 1),
        color: columnString(stmt, index: 2),
        pomodoroConfig: pomodoroConfig,
        createdAt: columnDate(stmt, index: 5),
        updatedAt: columnDate(stmt, index: 6)
      )
    }
  }

  // MARK: - Day Records

  func fetchDayRecord(date: String) async throws -> DayRecord? {
    // Try local cache first
    if let cached = try loadDayRecordFromCache(date: date) {
      print("[SQLite] Loaded day record for \(date) from cache")
      return cached
    }

    // Fetch from remote
    do {
      let records = try await remoteAPI.fetchDayRecords(from: date, to: date)
      if let record = records.first {
        // Cache it locally
        try await cacheDayRecord(record)
        return record
      }
      return nil
    } catch {
      print("[SQLite] Remote fetch failed for day record \(date): \(error)")
      return nil
    }
  }

  func createDayRecord(date: String) async throws -> DayRecord {
    // Create on remote
    let record = try await remoteAPI.createDayRecord(date: date)

    // Cache locally
    try await cacheDayRecord(record)

    return record
  }

  private func loadDayRecordFromCache(date: String) throws -> DayRecord? {
    let sql =
      "SELECT id, calendar_date, review_status, created_at, updated_at FROM day_records WHERE calendar_date = ?"

    guard
      let record = try queryOne(
        sql, params: [date],
        mapper: { stmt in
          return (
            id: columnInt(stmt, index: 0),
            calendarDate: columnString(stmt, index: 1),
            reviewStatus: columnString(stmt, index: 2),
            createdAt: columnDate(stmt, index: 3),
            updatedAt: columnDate(stmt, index: 4)
          )
        })
    else {
      return nil
    }

    // Load actual blocks
    let actualBlocksSql = "SELECT * FROM actual_blocks WHERE day_record_id = ? ORDER BY id"
    let actualBlocks = try query(actualBlocksSql, params: [record.id]) { stmt in
      return ActualBlock(
        id: columnInt(stmt, index: 0),
        categoryId: columnIntOptional(stmt, index: 2),
        blockType: columnString(stmt, index: 3),
        startTime: columnString(stmt, index: 4),
        durationMinutes: columnInt(stmt, index: 5)
      )
    }

    return DayRecord(
      id: record.id,
      calendarDate: record.calendarDate,
      reviewStatus: record.reviewStatus,
      actualBlocks: actualBlocks,
      createdAt: record.createdAt,
      updatedAt: record.updatedAt
    )
  }

  private func cacheDayRecord(_ record: DayRecord) async throws {
    try execute("BEGIN TRANSACTION")
    do {
      let iso8601 = ISO8601DateFormatter()

      // Upsert day record
      let upsertSql = """
        INSERT INTO day_records (id, calendar_date, review_status, created_at, updated_at)
        VALUES (?, ?, ?, ?, ?)
        ON CONFLICT(id) DO UPDATE SET
          calendar_date = excluded.calendar_date,
          review_status = excluded.review_status,
          created_at = excluded.created_at,
          updated_at = excluded.updated_at
        """
      try execute(
        upsertSql,
        params: [
          record.id,
          record.calendarDate,
          record.reviewStatus,
          iso8601.string(from: record.createdAt),
          iso8601.string(from: record.updatedAt),
        ])

      // Delete existing actual blocks
      try execute("DELETE FROM actual_blocks WHERE day_record_id = ?", params: [record.id])

      // Insert actual blocks
      for block in record.actualBlocks {
        let insertSql = """
          INSERT INTO actual_blocks (id, day_record_id, category_id, block_type, start_time, duration_minutes)
          VALUES (?, ?, ?, ?, ?, ?)
          """
        try execute(
          insertSql,
          params: [
            block.id, record.id, block.categoryId as Any, block.blockType, block.startTime,
            block.durationMinutes,
          ])
      }

      try execute("COMMIT")
    } catch {
      try execute("ROLLBACK")
      throw error
    }
  }

  // MARK: - Events & Reality Logging

  func submitEvents(dayRecordId: Int, events: [DayEvent]) async throws -> DayEventsResponse {
    // Store events as pending locally first
    try storePendingEvents(dayRecordId: dayRecordId, events: events)

    // Try to sync immediately
    if remoteAPI.hasAuthToken() {
      do {
        let response = try await remoteAPI.postDayEvents(dayRecordId: dayRecordId, events: events)

        // Mark events as synced
        try markEventsAsSynced(dayRecordId: dayRecordId, events: events)

        // Update cached actual blocks
        try await updateActualBlocksInCache(
          dayRecordId: dayRecordId, actualBlocks: response.actualBlocks)

        return response
      } catch {
        print("[SQLite] Failed to sync events immediately: \(error)")
        // Return a local response (actual blocks will be computed on next sync)
        return DayEventsResponse(
          createdEvents: events.map { event in
            CreatedEvent(
              id: -1,  // Temporary ID
              eventType: event.eventType,
              outgoingCategoryId: event.outgoingCategoryId,
              incomingCategoryId: event.incomingCategoryId,
              occurredAt: event.occurredAt
            )
          },
          actualBlocks: []  // Will be computed after sync
        )
      }
    }

    // Offline mode - return optimistic response
    return DayEventsResponse(
      createdEvents: events.map { event in
        CreatedEvent(
          id: -1,
          eventType: event.eventType,
          outgoingCategoryId: event.outgoingCategoryId,
          incomingCategoryId: event.incomingCategoryId,
          occurredAt: event.occurredAt
        )
      },
      actualBlocks: []
    )
  }

  private func storePendingEvents(dayRecordId: Int, events: [DayEvent]) throws {
    let iso8601 = ISO8601DateFormatter()

    for event in events {
      let sql = """
        INSERT INTO pending_events (day_record_id, event_type, outgoing_category_id, incoming_category_id, occurred_at, synced)
        VALUES (?, ?, ?, ?, ?, 0)
        """
      try execute(
        sql,
        params: [
          dayRecordId,
          event.eventType,
          event.outgoingCategoryId as Any,
          event.incomingCategoryId as Any,
          iso8601.string(from: event.occurredAt),
        ])
    }
    print("[SQLite] Stored \(events.count) pending events for day record \(dayRecordId)")
  }

  private func markEventsAsSynced(dayRecordId: Int, events: [DayEvent]) throws {
    let sql = "UPDATE pending_events SET synced = 1 WHERE day_record_id = ? AND synced = 0"
    try execute(sql, params: [dayRecordId])
    print("[SQLite] Marked events as synced for day record \(dayRecordId)")
  }

  private func updateActualBlocksInCache(dayRecordId: Int, actualBlocks: [ActualBlock]) async throws
  {
    try execute("BEGIN TRANSACTION")
    do {
      // Delete existing actual blocks
      try execute("DELETE FROM actual_blocks WHERE day_record_id = ?", params: [dayRecordId])

      // Insert new actual blocks
      for block in actualBlocks {
        let sql = """
          INSERT INTO actual_blocks (id, day_record_id, category_id, block_type, start_time, duration_minutes)
          VALUES (?, ?, ?, ?, ?, ?)
          """
        try execute(
          sql,
          params: [
            block.id, dayRecordId, block.categoryId as Any, block.blockType, block.startTime,
            block.durationMinutes,
          ])
      }

      try execute("COMMIT")
    } catch {
      try execute("ROLLBACK")
      throw error
    }
  }

  // MARK: - Schedule

  func fetchTodaySchedule() async throws -> TodaySchedule {
    return try await remoteAPI.fetchTodaySchedule()
  }

  // MARK: - Sync & Persistence

  func hasPendingSync() async -> Bool {
    do {
      let sql = "SELECT COUNT(*) FROM pending_events WHERE synced = 0"
      let count =
        try queryOne(
          sql,
          mapper: { stmt in
            return columnInt(stmt, index: 0)
          }) ?? 0
      return count > 0
    } catch {
      print("[SQLite] Error checking pending sync: \(error)")
      return false
    }
  }

  func synchronize() async throws {
    guard remoteAPI.hasAuthToken() else {
      print("[SQLite] Cannot sync: not authenticated")
      return
    }

    // Get all unsynced events grouped by day record
    let sql = """
      SELECT day_record_id, event_type, outgoing_category_id, incoming_category_id, occurred_at
      FROM pending_events
      WHERE synced = 0
      ORDER BY day_record_id, occurred_at ASC
      """

    let pendingEvents: [(dayRecordId: Int, event: DayEvent)] = try query(sql) { stmt in
      let iso8601 = ISO8601DateFormatter()
      return (
        dayRecordId: columnInt(stmt, index: 0),
        event: DayEvent(
          eventType: columnString(stmt, index: 1),
          outgoingCategoryId: columnIntOptional(stmt, index: 2),
          incomingCategoryId: columnIntOptional(stmt, index: 3),
          occurredAt: iso8601.date(from: columnString(stmt, index: 4)) ?? Date()
        )
      )
    }

    // Group by day record ID
    var eventsByDayRecord: [Int: [DayEvent]] = [:]
    for (dayRecordId, event) in pendingEvents {
      if eventsByDayRecord[dayRecordId] == nil {
        eventsByDayRecord[dayRecordId] = []
      }
      eventsByDayRecord[dayRecordId]?.append(event)
    }

    print("[SQLite] Syncing \(eventsByDayRecord.count) day records with pending events")

    // Sync each day record's events
    for (dayRecordId, events) in eventsByDayRecord {
      do {
        let response = try await remoteAPI.postDayEvents(dayRecordId: dayRecordId, events: events)

        // Mark events as synced
        try markEventsAsSynced(dayRecordId: dayRecordId, events: events)

        // Update actual blocks
        try await updateActualBlocksInCache(
          dayRecordId: dayRecordId, actualBlocks: response.actualBlocks)

        print("[SQLite] Successfully synced \(events.count) events for day record \(dayRecordId)")
      } catch {
        print("[SQLite] Failed to sync day record \(dayRecordId): \(error)")
        throw error
      }
    }
  }

  // MARK: - Utilities

  private func clearLocalData() throws {
    try execute("BEGIN TRANSACTION")
    do {
      try execute("DELETE FROM categories")
      try execute("DELETE FROM pending_events")
      try execute("DELETE FROM actual_blocks")
      try execute("DELETE FROM day_records")
      try execute("COMMIT")
      print("[SQLite] Cleared all local data")
    } catch {
      try execute("ROLLBACK")
      throw error
    }
  }
}
