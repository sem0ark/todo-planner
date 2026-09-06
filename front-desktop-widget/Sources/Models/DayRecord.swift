import Foundation

struct SnapshotBlock: Identifiable, Codable {
  let id: Int
  let snapshotId: Int?
  let categoryId: Int
  let startTime: String  // HH:MM format
  let durationMinutes: Int

  enum CodingKeys: String, CodingKey {
    case id
    case snapshotId = "snapshot_id"
    case categoryId = "category_id"
    case startTime = "start_time"
    case durationMinutes = "duration_minutes"
  }

  init(from decoder: Decoder) throws {
    let container = try decoder.container(keyedBy: CodingKeys.self)
    id = try container.decodeIfPresent(Int.self, forKey: .id) ?? 0
    snapshotId = try container.decodeIfPresent(Int.self, forKey: .snapshotId)
    categoryId = try container.decode(Int.self, forKey: .categoryId)
    startTime = try container.decode(String.self, forKey: .startTime)
    durationMinutes = try container.decode(Int.self, forKey: .durationMinutes)
  }

  init(
    id: Int, snapshotId: Int? = nil, categoryId: Int,
    startTime: String, durationMinutes: Int
  ) {
    self.id = id
    self.snapshotId = snapshotId
    self.categoryId = categoryId
    self.startTime = startTime
    self.durationMinutes = durationMinutes
  }
}

typealias PlannedBlock = SnapshotBlock

struct ActualBlock: Identifiable, Codable {
  let id: Int
  let dayRecordId: Int?
  let categoryId: Int?
  let blockType: String  // "actual" | "blank" | "untracked"
  let startTime: String  // HH:MM format
  let durationMinutes: Int
  let isOpen: Bool

  enum CodingKeys: String, CodingKey {
    case id
    case dayRecordId = "day_record_id"
    case categoryId = "category_id"
    case blockType = "block_type"
    case startTime = "start_time"
    case durationMinutes = "duration_minutes"
    case isOpen = "is_open"
    case updatedAt = "updated_at"
  }

  let updatedAt: Date?

  init(from decoder: Decoder) throws {
    let container = try decoder.container(keyedBy: CodingKeys.self)
    id = try container.decodeIfPresent(Int.self, forKey: .id) ?? 0
    dayRecordId = try container.decodeIfPresent(Int.self, forKey: .dayRecordId)
    categoryId = try container.decodeIfPresent(Int.self, forKey: .categoryId)
    blockType = try container.decode(String.self, forKey: .blockType)
    startTime = try container.decode(String.self, forKey: .startTime)
    durationMinutes = try container.decode(Int.self, forKey: .durationMinutes)
    isOpen = try container.decodeIfPresent(Bool.self, forKey: .isOpen) ?? false
    updatedAt = try container.decodeIfPresent(Date.self, forKey: .updatedAt)
  }

  init(
    id: Int, dayRecordId: Int? = nil, categoryId: Int?, blockType: String,
    startTime: String, durationMinutes: Int, isOpen: Bool = false, updatedAt: Date? = nil
  ) {
    self.id = id
    self.dayRecordId = dayRecordId
    self.categoryId = categoryId
    self.blockType = blockType
    self.startTime = startTime
    self.durationMinutes = durationMinutes
    self.isOpen = isOpen
    self.updatedAt = updatedAt
  }
}

struct ActualBlockInput: Encodable {
  let categoryId: Int?
  let blockType: String
  let startTime: String
  let durationMinutes: Int

  enum CodingKeys: String, CodingKey {
    case categoryId = "category_id"
    case blockType = "block_type"
    case startTime = "start_time"
    case durationMinutes = "duration_minutes"
  }
}

struct DayRecordUpdateResponse: Decodable {
  let actualBlocks: [ActualBlock]
  let updatedAt: Date

  enum CodingKeys: String, CodingKey {
    case actualBlocks = "actual_blocks"
    case updatedAt = "updated_at"
  }
}

struct DayRecord: Identifiable, Codable {
  let id: Int
  let calendarDate: String  // YYYY-MM-DD
  let dayTemplateId: Int?
  let snapshotId: Int?
  let snapshotBlocks: [PlannedBlock]
  let actualBlocks: [ActualBlock]
  let createdAt: Date
  let updatedAt: Date

  enum CodingKeys: String, CodingKey {
    case id
    case calendarDate = "calendar_date"
    case dayTemplateId = "day_template_id"
    case snapshotId = "snapshot_id"
    case snapshotBlocks = "snapshot_blocks"
    case snapshot
    case actualBlocks = "actual_blocks"
    case createdAt = "created_at"
    case updatedAt = "updated_at"
  }

  init(
    id: Int, calendarDate: String, reviewStatus: String? = nil,
    dayTemplateId: Int? = nil, snapshotId: Int? = nil,
    snapshotBlocks: [PlannedBlock] = [], actualBlocks: [ActualBlock],
    createdAt: Date, updatedAt: Date
  ) {
    self.id = id
    self.calendarDate = calendarDate
    self.dayTemplateId = dayTemplateId
    self.snapshotId = snapshotId
    self.snapshotBlocks = snapshotBlocks
    self.actualBlocks = actualBlocks
    self.createdAt = createdAt
    self.updatedAt = updatedAt
  }

  init(from decoder: Decoder) throws {
    let container = try decoder.container(keyedBy: CodingKeys.self)
    id = try container.decodeIfPresent(Int.self, forKey: .id) ?? 0
    calendarDate = try container.decode(String.self, forKey: .calendarDate)
    dayTemplateId = try container.decodeIfPresent(Int.self, forKey: .dayTemplateId)
    if let directSnapshotId = try container.decodeIfPresent(Int.self, forKey: .snapshotId) {
      snapshotId = directSnapshotId
      snapshotBlocks =
        try container.decodeIfPresent([PlannedBlock].self, forKey: .snapshotBlocks) ?? []
    } else if let snapshot = try container.decodeIfPresent(DaySnapshot.self, forKey: .snapshot) {
      snapshotId = snapshot.snapshotId
      snapshotBlocks = snapshot.blocks.enumerated().map { index, block in
        SnapshotBlock(
          id: index, categoryId: block.categoryId, startTime: block.startTime,
          durationMinutes: block.durationMinutes)
      }
    } else {
      snapshotId = nil
      snapshotBlocks = []
    }
    actualBlocks = try container.decodeIfPresent([ActualBlock].self, forKey: .actualBlocks) ?? []
    createdAt = try container.decode(Date.self, forKey: .createdAt)
    updatedAt = try container.decode(Date.self, forKey: .updatedAt)
  }

  func encode(to encoder: Encoder) throws {
    var container = encoder.container(keyedBy: CodingKeys.self)
    try container.encode(id, forKey: .id)
    try container.encode(calendarDate, forKey: .calendarDate)
    try container.encodeIfPresent(dayTemplateId, forKey: .dayTemplateId)
    try container.encodeIfPresent(snapshotId, forKey: .snapshotId)
    try container.encode(snapshotBlocks, forKey: .snapshotBlocks)
    try container.encode(actualBlocks, forKey: .actualBlocks)
    try container.encode(createdAt, forKey: .createdAt)
    try container.encode(updatedAt, forKey: .updatedAt)
  }
}

private struct DaySnapshot: Decodable {
  let snapshotId: Int
  let blocks: [SnapshotBlock]
  enum CodingKeys: String, CodingKey {
    case snapshotId = "snapshot_id"
    case blocks
  }
}

struct DayRecordsResponse: Codable {
  let dayRecords: [DayRecord]

  enum CodingKeys: String, CodingKey {
    case dayRecords = "day_records"
  }
}
