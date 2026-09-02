import Foundation

struct TodaySchedule: Codable {
  let calendarDate: String
  let dayTemplateId: Int?
  let template: DayTemplate?

  enum CodingKeys: String, CodingKey {
    case calendarDate = "calendar_date"
    case dayTemplateId = "day_template_id"
    case template
  }
}

struct DayTemplate: Codable {
  let id: Int
  let name: String
  let templateGroupId: Int?
  let plannedBlocks: [PlannedBlock]

  enum CodingKeys: String, CodingKey {
    case id
    case name
    case templateGroupId = "template_group_id"
    case plannedBlocks = "planned_blocks"
  }
}
