import SwiftUI

struct Palette {
    // Background colors
    static let deepVoid = "#001a24"        // Outer background
    static let baseVoid = "#003448"        // Widget surface

    // Text colors
    static let primaryText = "#ffffff"     // White text
    static let mutedText = "#dee2ef"       // Light gray text
    static let secondaryText = "#91a6be"   // Medium gray text
    static let structuralBorder = "#afb6cf" // Border color

    // State-specific colors
    static let offsetGreen = "#10b981"     // Offset bar highlight (State 3)
    static let dashedBorder = "#dee2ef"    // Dashed separator line

    // Opacity values
    static let subtleLineOpacity = 0.05    // Very subtle divider lines
    static let overlayOpacity = 0.1        // Subtle overlays
    static let borderOpacity = 0.2         // Borders and dividers
    static let hoverOpacity = 0.3          // Hover states
    static let shadowOpacity = 0.5         // Drop shadows
    static let labelOpacity = 0.7          // Secondary labels
    static let iconOpacity = 0.4           // Muted icons
    static let progressBgOpacity = 0.2     // Progress bar backgrounds
    static let progressFillOpacity = 0.6   // Progress bar fills
    static let breatheMinOpacity = 0.85    // Animation minimum
}

struct StyleTokens {
    // Background colors
    static let deepVoid = Color(hex: Palette.deepVoid)
    static let baseVoid = Color(hex: Palette.baseVoid)

    // Text colors
    static let primaryText = Color(hex: Palette.primaryText)
    static let mutedText = Color(hex: Palette.mutedText)
    static let secondaryText = Color(hex: Palette.secondaryText)
    static let structuralBorder = Color(hex: Palette.structuralBorder)

    // State colors
    static let offsetGreen = Color(hex: Palette.offsetGreen)

    // Border radius
    static let radiusOuter: CGFloat = 12              // Widget outer corners
    static let radiusInner: CGFloat = 4               // Inner elements
    static let radiusButton: CGFloat = 4              // Small buttons
}

struct ContentView: View {
    @StateObject private var widgetState = WidgetState()
    @State private var isAuthenticated: Bool = false

    var body: some View {
        Group {
            if isAuthenticated {
                HStack(spacing: 0) {
                    // Left Panel (65%)
                    LeftPanelView(widgetState: widgetState)
                        .frame(width: 208)
                        .overlay(
                            Rectangle()
                                .fill(StyleTokens.structuralBorder.opacity(Palette.overlayOpacity))
                                .frame(width: 1),
                            alignment: .trailing
                        )

                    // Right Rail (35%)
                    RightRailView(widgetState: widgetState)
                        .frame(width: 112)
                }
                .task {
                    await widgetState.initialize()
                    widgetState.startPeriodicRefresh()
                }
                .onDisappear {
                    widgetState.stopPeriodicRefresh()
                }
                .keyboardShortcuts(widgetState: widgetState)
            } else {
                LoginView(isAuthenticated: $isAuthenticated)
            }
        }
        .frame(width: 320, height: 200)
        .background(StyleTokens.baseVoid)
        .cornerRadius(StyleTokens.radiusOuter)
        .overlay(
            RoundedRectangle(cornerRadius: StyleTokens.radiusOuter)
                .stroke(StyleTokens.structuralBorder.opacity(Palette.borderOpacity), lineWidth: 1)
        )
        .shadow(color: .black.opacity(Palette.shadowOpacity), radius: 12, x: 0, y: 4)
    }
}

struct LeftPanelView: View {
    @ObservedObject var widgetState: WidgetState

    var body: some View {
        ZStack {
            switch widgetState.displayState {
            case .confirmationPrompt:
                ConfirmationPromptView(widgetState: widgetState)
            case .active:
                ActiveView(widgetState: widgetState)
            case .offSchedule:
                OffScheduleView(widgetState: widgetState)
            }
        }
    }
}

// State 1: Confirmation Prompt
struct ConfirmationPromptView: View {
    @ObservedObject var widgetState: WidgetState
    @State private var breatheScale: CGFloat = 1.0
    @State private var breatheOpacity: Double = 1.0

    var body: some View {
        Button(action: {
            Task {
                await widgetState.confirmPlanned()
            }
        }) {
            ZStack(alignment: .topLeading) {
                // Category color background with breathe animation
                Color(hex: widgetState.plannedCategory?.color ?? "#808080")
                    .scaleEffect(breatheScale)
                    .opacity(breatheOpacity)

                VStack(alignment: .leading, spacing: 4) {
                    // Top label with alert icon
                    HStack(spacing: 4) {
                        Text("ACTUAL")
                            .font(.system(size: 10, weight: .bold, design: .default))
                            .foregroundColor(.white.opacity(Palette.labelOpacity))
                            .tracking(0.5)

                        Spacer()

                        // Alert icon (bouncing)
                        Circle()
                            .fill(.white.opacity(Palette.labelOpacity))
                            .frame(width: 12, height: 12)
                            .overlay {
                                Image(systemName: "exclamationmark.circle")
                                    .font(.system(size: 10))
                                    .foregroundColor(Color(hex: widgetState.plannedCategory?.color ?? "#808080"))
                            }
                    }

                    Spacer()

                    // Category name (large, bold, uppercase)
                    Text(widgetState.plannedCategory?.name.uppercased() ?? "")
                        .font(.system(size: 20, weight: .black, design: .default))
                        .foregroundColor(.white)
                        .lineLimit(2)
                }
                .padding(.horizontal, 12)
                .padding(.top, 12)
                .padding(.bottom, 12)
            }
        }
        .buttonStyle(PlainButtonStyle())
        .onAppear {
            // Breathe animation (2s cycle, matching React)
            withAnimation(.easeInOut(duration: 2.0).repeatForever(autoreverses: true)) {
                breatheScale = 1.01
                breatheOpacity = Palette.breatheMinOpacity
            }
        }
    }
}

// State 2: Active/On-Schedule
struct ActiveView: View {
    @ObservedObject var widgetState: WidgetState

    var body: some View {
        if let category = widgetState.currentCategory {
            VStack(spacing: 0) {
                // Main category block
                ZStack(alignment: .topLeading) {
                    Color(hex: category.color)

                    VStack(alignment: .leading, spacing: 4) {
                        // Top label with checkmark
                        HStack(spacing: 4) {
                            Text("ACTUAL")
                                .font(.system(size: 10, weight: .bold, design: .default))
                                .foregroundColor(.white.opacity(Palette.labelOpacity))
                                .tracking(0.5)

                            Spacer()

                            if widgetState.isConfirmed {
                                Image(systemName: "checkmark.circle")
                                    .font(.system(size: 12))
                                    .foregroundColor(.white.opacity(Palette.iconOpacity))
                            }
                        }

                        Spacer()

                        // Category name (large, bold, uppercase)
                        Text(category.name.uppercased())
                            .font(.system(size: 20, weight: .black, design: .default))
                            .foregroundColor(.white)
                            .lineLimit(2)

                        Spacer()

                        // Progress bar container
                        VStack(alignment: .leading, spacing: 4) {
                            // Progress bar
                            GeometryReader { geometry in
                                ZStack(alignment: .leading) {
                                    // Background
                                    RoundedRectangle(cornerRadius: 2)
                                        .fill(.black.opacity(Palette.progressBgOpacity))
                                        .frame(height: 4)

                                    // Progress fill
                                    RoundedRectangle(cornerRadius: 2)
                                        .fill(.white.opacity(Palette.progressFillOpacity))
                                        .frame(width: geometry.size.width * widgetState.progressPercentage, height: 4)
                                }
                            }
                            .frame(height: 4)

                            // Elapsed time
                            Text("\(Int(widgetState.progressPercentage * Double(widgetState.plannedDurationMinutes)))m elapsed")
                                .font(.system(size: 9, design: .monospaced))
                                .monospacedDigit()
                                .foregroundColor(.white.opacity(Palette.progressFillOpacity))
                        }
                    }
                    .padding(.horizontal, 12)
                    .padding(.vertical, 12)
                }
            }
        } else {
            // Loading or no category selected
            ZStack {
                StyleTokens.baseVoid

                VStack(spacing: 8) {
                    Text("Select a category")
                        .font(.system(size: 14, weight: .medium))
                        .foregroundColor(StyleTokens.mutedText)

                    Text("Press 1-9 to start")
                        .font(.system(size: 10))
                        .foregroundColor(StyleTokens.mutedText.opacity(Palette.labelOpacity))
                }
            }
        }
    }
}

// State 3: Off-Schedule
struct OffScheduleView: View {
    @ObservedObject var widgetState: WidgetState
    @State private var pulseOpacity: Double = 1.0

    var body: some View {
        VStack(spacing: 0) {
            // Offset bar (conditional visibility) - Top
            if widgetState.showOffsetBar {
                HStack(spacing: 8) {
                    Text("T-\(widgetState.offsetMinutes)m")
                        .font(.system(size: 9, weight: .bold, design: .monospaced))
                        .monospacedDigit()
                        .foregroundColor(StyleTokens.offsetGreen)

                    Spacer()

                    // Offset buttons
                    HStack(spacing: 4) {
                        OffsetButton(label: "+5m", minutes: 5, widgetState: widgetState)
                        OffsetButton(label: "+15m", minutes: 15, widgetState: widgetState)
                    }
                }
                .padding(.horizontal, 8)
                .padding(.vertical, 4)
                .frame(height: 28)
                .background(StyleTokens.offsetGreen.opacity(Palette.progressBgOpacity))
                .overlay(
                    Rectangle()
                        .fill(StyleTokens.offsetGreen.opacity(Palette.hoverOpacity))
                        .frame(height: 1),
                    alignment: .bottom
                )
            }

            // Current (Actual) - Top section
            ZStack(alignment: .topLeading) {
                Color(hex: widgetState.currentCategory?.color ?? "#808080")

                VStack(alignment: .leading, spacing: 4) {
                    Text("ACTUAL")
                        .font(.system(size: 10, weight: .bold, design: .default))
                        .foregroundColor(.white.opacity(Palette.labelOpacity))
                        .tracking(0.5)

                    Spacer()

                    Text(widgetState.currentCategory?.name.uppercased() ?? "")
                        .font(.system(size: 20, weight: .black, design: .default))
                        .foregroundColor(.white)
                        .lineLimit(2)

                    Spacer()
                }
                .padding(.horizontal, 12)
                .padding(.vertical, 12)
            }
            .frame(height: widgetState.showOffsetBar ? 86 : 100)

            // Planned - Bottom section (pulsing clickable with dashed border)
            Button(action: {
                Task {
                    await widgetState.syncToPlan()
                }
            }) {
                ZStack(alignment: .topLeading) {
                    Color(hex: widgetState.plannedCategory?.color ?? "#808080")
                        .opacity(pulseOpacity)

                    VStack(alignment: .leading, spacing: 4) {
                        HStack {
                            Text("PLANNED")
                                .font(.system(size: 10, weight: .bold, design: .default))
                                .foregroundColor(.white.opacity(Palette.labelOpacity))
                                .tracking(0.5)

                            Spacer()

                            Text("RETURN ↵")
                                .font(.system(size: 9, design: .monospaced))
                                .foregroundColor(.white)
                                .padding(.horizontal, 4)
                                .padding(.vertical, 2)
                                .background(.black.opacity(Palette.progressBgOpacity))
                                .cornerRadius(2)
                        }

                        Spacer()

                        Text(widgetState.plannedCategory?.name.uppercased() ?? "")
                            .font(.system(size: 14, weight: .bold, design: .default))
                            .foregroundColor(.white)

                        Spacer()

                        // Time remaining progress bar
                        GeometryReader { geometry in
                            ZStack(alignment: .leading) {
                                RoundedRectangle(cornerRadius: 2)
                                    .fill(.black.opacity(Palette.progressBgOpacity))
                                    .frame(height: 4)

                                RoundedRectangle(cornerRadius: 2)
                                    .fill(.white)
                                    .frame(width: geometry.size.width * (1.0 - widgetState.progressPercentage), height: 4)
                            }
                        }
                        .frame(height: 4)
                    }
                    .padding(.horizontal, 12)
                    .padding(.vertical, 12)
                }
                // Dashed border at top
                .overlay(
                    DashedLine()
                        .stroke(StyleTokens.mutedText, style: StrokeStyle(lineWidth: 2, dash: [4, 4]))
                        .frame(height: 2),
                    alignment: .top
                )
            }
            .buttonStyle(PlainButtonStyle())
            .frame(height: widgetState.showOffsetBar ? 86 : 100)
            .onAppear {
                withAnimation(.easeInOut(duration: 0.8).repeatForever(autoreverses: true)) {
                    pulseOpacity = Palette.breatheMinOpacity
                }
            }
        }
    }
}

// Helper view for offset buttons
struct OffsetButton: View {
    let label: String
    let minutes: Int
    let widgetState: WidgetState

    var body: some View {
        Button(action: {
            Task { await widgetState.adjustOffset(minutes: minutes) }
        }) {
            Text(label)
                .font(.system(size: 9, design: .monospaced))
                .foregroundColor(.white)
                .padding(.horizontal, 6)
                .padding(.vertical, 2)
                .background(StyleTokens.baseVoid)
                .overlay(
                    RoundedRectangle(cornerRadius: 2)
                        .stroke(StyleTokens.offsetGreen.opacity(Palette.iconOpacity), lineWidth: 1)
                )
                .cornerRadius(2)
        }
        .buttonStyle(PlainButtonStyle())
    }
}

// Dashed line shape
struct DashedLine: Shape {
    func path(in rect: CGRect) -> Path {
        var path = Path()
        path.move(to: CGPoint(x: rect.minX, y: rect.minY))
        path.addLine(to: CGPoint(x: rect.maxX, y: rect.minY))
        return path
    }
}

struct RightRailView: View {
    @ObservedObject var widgetState: WidgetState

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            ForEach(Array(widgetState.categories.enumerated()), id: \.element.id) { index, category in
                CategoryRow(index: index + 1, category: category, widgetState: widgetState)
            }
            Spacer()
        }
        .background(StyleTokens.baseVoid)
    }
}

struct CategoryRow: View {
    let index: Int
    let category: Category
    @ObservedObject var widgetState: WidgetState

    private var isActive: Bool {
        widgetState.currentCategory?.id == category.id
    }

    var body: some View {
        Button(action: {
            Task {
                await widgetState.transitionToCategory(category)
            }
        }) {
            HStack(spacing: 8) {
                // Color indicator dot with glow when active
                Circle()
                    .fill(Color(hex: category.color))
                    .frame(width: 6, height: 6)
                    .shadow(color: isActive ? Color(hex: category.color) : .clear, radius: 4)

                // Category name
                Text(category.name.uppercased())
                    .font(.system(size: 10, weight: .bold, design: .default))
                    .foregroundColor(isActive ? .white : StyleTokens.secondaryText)
                    .tracking(1.2)
                    .lineLimit(1)
                    .truncationMode(.tail)

                Spacer(minLength: 0)

                // Index number
                Text("\(index)")
                    .font(.system(size: 9, design: .monospaced))
                    .monospacedDigit()
                    .foregroundColor(StyleTokens.structuralBorder.opacity(Palette.iconOpacity))
            }
            .padding(.horizontal, 12)
            .padding(.vertical, 10)
            .contentShape(Rectangle())
        }
        .buttonStyle(PlainButtonStyle())
        .background(
            isActive
                ? StyleTokens.secondaryText.opacity(Palette.progressBgOpacity)
                : Color.clear
        )
        .overlay(
            Rectangle()
                .fill(StyleTokens.structuralBorder.opacity(Palette.subtleLineOpacity))
                .frame(height: 1),
            alignment: .bottom
        )
    }
}

extension Color {
    init(hex: String) {
        let hex = hex.trimmingCharacters(in: CharacterSet.alphanumerics.inverted)
        var int: UInt64 = 0
        Scanner(string: hex).scanHexInt64(&int)
        let a, r, g, b: UInt64
        switch hex.count {
        case 3: // RGB (12-bit)
            (a, r, g, b) = (255, (int >> 8) * 17, (int >> 4 & 0xF) * 17, (int & 0xF) * 17)
        case 6: // RGB (24-bit)
            (a, r, g, b) = (255, int >> 16, int >> 8 & 0xFF, int & 0xFF)
        case 8: // ARGB (32-bit)
            (a, r, g, b) = (int >> 24, int >> 16 & 0xFF, int >> 8 & 0xFF, int & 0xFF)
        default:
            (a, r, g, b) = (255, 0, 0, 0)
        }

        self.init(
            .sRGB,
            red: Double(r) / 255,
            green: Double(g) / 255,
            blue:  Double(b) / 255,
            opacity: Double(a) / 255
        )
    }
}

extension View {
    func fontVariantMonospacedDigit() -> some View {
        self.monospacedDigit()
    }
}

extension View {
    func keyboardShortcuts(widgetState: WidgetState) -> some View {
        self
            .onKeyPress(.space) {
                Task {
                    await widgetState.confirmPlanned()
                }
                return .handled
            }
            .onKeyPress(.return) {
                Task {
                    await widgetState.syncToPlan()
                }
                return .handled
            }
            .onKeyPress("1") { handleNumberKey(1, widgetState: widgetState) }
            .onKeyPress("2") { handleNumberKey(2, widgetState: widgetState) }
            .onKeyPress("3") { handleNumberKey(3, widgetState: widgetState) }
            .onKeyPress("4") { handleNumberKey(4, widgetState: widgetState) }
            .onKeyPress("5") { handleNumberKey(5, widgetState: widgetState) }
            .onKeyPress("6") { handleNumberKey(6, widgetState: widgetState) }
            .onKeyPress("7") { handleNumberKey(7, widgetState: widgetState) }
            .onKeyPress("8") { handleNumberKey(8, widgetState: widgetState) }
            .onKeyPress("9") { handleNumberKey(9, widgetState: widgetState) }
            .onKeyPress("[") {
                Task {
                    await widgetState.adjustOffset(minutes: -5)
                }
                return .handled
            }
            .onKeyPress("]") {
                Task {
                    await widgetState.adjustOffset(minutes: 5)
                }
                return .handled
            }
    }

    private func handleNumberKey(_ number: Int, widgetState: WidgetState) -> KeyPress.Result {
        let index = number - 1
        guard index < widgetState.categories.count else { return .ignored }

        Task {
            await widgetState.transitionToCategory(widgetState.categories[index])
        }
        return .handled
    }
}
