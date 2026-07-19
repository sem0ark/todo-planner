import Foundation

struct Category: Identifiable, Codable {
    let id: Int
    let name: String
    let color: String // Hex color
    let createdAt: Date
    let updatedAt: Date

    enum CodingKeys: String, CodingKey {
        case id
        case name
        case color
        case createdAt = "created_at"
        case updatedAt = "updated_at"
    }
}

struct CategoriesResponse: Codable {
    let categories: [Category]
}
