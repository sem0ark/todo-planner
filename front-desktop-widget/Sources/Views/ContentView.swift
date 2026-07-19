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

struct Typography {
    // Font sizes (minimum 10px everywhere)
    static let categoryNameLarge: CGFloat = 20        // Category name in main blocks
    static let categoryNameMedium: CGFloat = 14       // Category name in secondary blocks
    static let bodyMedium: CGFloat = 14               // Body text
    static let labelBold: CGFloat = 10                // Small labels (ACTUAL, PLANNED)
    static let smallMono: CGFloat = 10                // Small monospaced text (times, numbers)
    static let tinyMono: CGFloat = 10                 // Tiny monospaced text (was 9, now 10 minimum)
    static let categoryRowName: CGFloat = 10          // Right rail category names
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
    @State private var isCheckingAuth: Bool = true

    var body: some View {
        Group {
            if isCheckingAuth {
                // Show loading state while checking authentication
                ZStack {
                    StyleTokens.baseVoid
                    VStack(spacing: 8) {
                        ProgressView()
                            .progressViewStyle(CircularProgressViewStyle(tint: .white))
                        Text("Checking authentication...")
                            .font(.system(size: Typography.labelBold))
                            .foregroundColor(StyleTokens.mutedText)
                    }
                }
            } else if isAuthenticated {
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
                    RightRailView(widgetState: widgetState, isAuthenticated: $isAuthenticated)
                        .frame(width: 112)
                }
                .task {
                    await widgetState.initialize()
                    widgetState.startPeriodicRefresh()
                }
                .onDisappear {
                    widgetState.stopPeriodicRefresh()
                }
                .onReceive(NotificationCenter.default.publisher(for: .keyPressed)) { notification in
                    if let event = notification.object as? NSEvent {
                        handleKeyPress(event)
                    }
                }
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
        .task {
            await checkAuthentication()
        }
    }

    private func checkAuthentication() async {
        print("[AUTH] Checking for saved token...")

        if APIClient.shared.hasAuthToken() {
            print("[AUTH] Token found, validating...")
            let isValid = await APIClient.shared.validateToken()

            if isValid {
                print("[OK] Token is valid, user authenticated")
                isAuthenticated = true
            } else {
                print("[AUTH] Token invalid, showing login")
                isAuthenticated = false
            }
        } else {
            print("[AUTH] No token found, showing login")
            isAuthenticated = false
        }

        isCheckingAuth = false
    }

    private func handleKeyPress(_ event: NSEvent) {
        let key = event.charactersIgnoringModifiers ?? ""

        print("[KEY] Key pressed: '\(key)' (keyCode: \(event.keyCode))")

        Task {
            switch key {
            case " ":
                print("[KEY] Space - confirming planned")
                await widgetState.confirmPlanned()
            case "\r": // Return key
                print("[KEY] Return - syncing to plan")
                await widgetState.syncToPlan()
            case "1", "2", "3", "4", "5", "6", "7", "8", "9":
                if let number = Int(key) {
                    let index = number - 1
                    guard index < widgetState.categories.count else { return }
                    print("[KEY] Number \(number) - transitioning to category")
                    await widgetState.transitionToCategory(widgetState.categories[index])
                }
            case "[":
                print("[KEY] [ - adjusting offset -5m")
                await widgetState.adjustOffset(minutes: -5)
            case "]":
                print("[KEY] ] - adjusting offset +5m")
                await widgetState.adjustOffset(minutes: 5)
            default:
                break
            }
        }
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
        let categoryColor = widgetState.plannedCategory?.color ?? "#808080"
        let textColor = Color.contrastingTextColor(for: categoryColor)

        Button(action: {
            Task {
                await widgetState.confirmPlanned()
            }
        }) {
            ZStack(alignment: .topLeading) {
                // Category color background with breathe animation
                Color(hex: categoryColor)
                    .scaleEffect(breatheScale)
                    .opacity(breatheOpacity)

                VStack(alignment: .leading, spacing: 4) {
                    // Top label with alert icon
                    HStack(spacing: 4) {
                        Text("ACTUAL")
                            .font(.system(size: Typography.labelBold, weight: .bold, design: .default))
                            .foregroundColor(textColor.opacity(Palette.labelOpacity))
                            .tracking(0.5)

                        Spacer()

                        // Alert icon (bouncing)
                        Circle()
                            .fill(textColor.opacity(Palette.labelOpacity))
                            .frame(width: 12, height: 12)
                            .overlay {
                                Image(systemName: "exclamationmark.circle")
                                    .font(.system(size: Typography.labelBold))
                                    .foregroundColor(Color(hex: categoryColor))
                            }
                    }

                    Spacer()

                    // Category name (large, bold, uppercase)
                    Text(widgetState.plannedCategory?.name.uppercased() ?? "")
                        .font(.system(size: Typography.categoryNameLarge, weight: .black, design: .default))
                        .foregroundColor(textColor)
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
            let textColor = Color.contrastingTextColor(for: category.color)

            VStack(spacing: 0) {
                // Main category block
                ZStack(alignment: .topLeading) {
                    Color(hex: category.color)

                    VStack(alignment: .leading, spacing: 4) {
                        // Top label with checkmark
                        HStack(spacing: 4) {
                            Text("ACTUAL")
                                .font(.system(size: Typography.labelBold, weight: .bold, design: .default))
                                .foregroundColor(textColor.opacity(Palette.labelOpacity))
                                .tracking(0.5)

                            Spacer()

                            if widgetState.isConfirmed {
                                Image(systemName: "checkmark.circle")
                                    .font(.system(size: 12))
                                    .foregroundColor(textColor.opacity(Palette.iconOpacity))
                            }
                        }

                        Spacer()

                        // Category name (large, bold, uppercase)
                        Text(category.name.uppercased())
                            .font(.system(size: Typography.categoryNameLarge, weight: .black, design: .default))
                            .foregroundColor(textColor)
                            .lineLimit(2)

                        Spacer()

                        // Progress bar container
                        VStack(alignment: .leading, spacing: 4) {
                            // Progress bar
                            GeometryReader { geometry in
                                ZStack(alignment: .leading) {
                                    // Background
                                    RoundedRectangle(cornerRadius: 2)
                                        .fill(textColor.opacity(Palette.progressBgOpacity))
                                        .frame(height: 4)

                                    // Progress fill
                                    RoundedRectangle(cornerRadius: 2)
                                        .fill(textColor.opacity(Palette.progressFillOpacity))
                                        .frame(width: geometry.size.width * widgetState.progressPercentage, height: 4)
                                }
                            }
                            .frame(height: 4)

                            // Elapsed time
                            Text("\(Int(widgetState.progressPercentage * Double(widgetState.plannedDurationMinutes)))m elapsed")
                                .font(.system(size: Typography.tinyMono, design: .monospaced))
                                .monospacedDigit()
                                .foregroundColor(textColor.opacity(Palette.progressFillOpacity))
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
                        .font(.system(size: Typography.bodyMedium, weight: .medium))
                        .foregroundColor(StyleTokens.mutedText)

                    Text("Press 1-9 to start")
                        .font(.system(size: Typography.labelBold))
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
        let currentCategoryColor = widgetState.currentCategory?.color ?? "#808080"
        let plannedCategoryColor = widgetState.plannedCategory?.color ?? "#808080"
        let currentTextColor = Color.contrastingTextColor(for: currentCategoryColor)
        let plannedTextColor = Color.contrastingTextColor(for: plannedCategoryColor)

        VStack(spacing: 0) {
            // Offset bar (conditional visibility) - Top
            if widgetState.showOffsetBar {
                HStack(spacing: 8) {
                    Text("T-\(widgetState.offsetMinutes)m")
                        .font(.system(size: Typography.tinyMono, weight: .bold, design: .monospaced))
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
                Color(hex: currentCategoryColor)

                VStack(alignment: .leading, spacing: 4) {
                    Text("ACTUAL")
                        .font(.system(size: Typography.labelBold, weight: .bold, design: .default))
                        .foregroundColor(currentTextColor.opacity(Palette.labelOpacity))
                        .tracking(0.5)

                    Spacer()

                    Text(widgetState.currentCategory?.name.uppercased() ?? "")
                        .font(.system(size: Typography.categoryNameLarge, weight: .black, design: .default))
                        .foregroundColor(currentTextColor)
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
                    Color(hex: plannedCategoryColor)
                        .opacity(pulseOpacity)

                    VStack(alignment: .leading, spacing: 4) {
                        HStack {
                            Text("PLANNED")
                                .font(.system(size: Typography.labelBold, weight: .bold, design: .default))
                                .foregroundColor(plannedTextColor.opacity(Palette.labelOpacity))
                                .tracking(0.5)

                            Spacer()

                            Text("RETURN ↵")
                                .font(.system(size: Typography.tinyMono, design: .monospaced))
                                .foregroundColor(plannedTextColor)
                                .padding(.horizontal, 4)
                                .padding(.vertical, 2)
                                .background(plannedTextColor.opacity(Palette.progressBgOpacity))
                                .colorInvert()
                                .cornerRadius(2)
                        }

                        Spacer()

                        Text(widgetState.plannedCategory?.name.uppercased() ?? "")
                            .font(.system(size: Typography.categoryNameMedium, weight: .bold, design: .default))
                            .foregroundColor(plannedTextColor)

                        Spacer()

                        // Time remaining progress bar
                        GeometryReader { geometry in
                            ZStack(alignment: .leading) {
                                RoundedRectangle(cornerRadius: 2)
                                    .fill(plannedTextColor.opacity(Palette.progressBgOpacity))
                                    .frame(height: 4)

                                RoundedRectangle(cornerRadius: 2)
                                    .fill(plannedTextColor)
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
                .font(.system(size: Typography.tinyMono, design: .monospaced))
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
    @Binding var isAuthenticated: Bool

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            ForEach(Array(widgetState.categories.enumerated()), id: \.element.id) { index, category in
                CategoryRow(index: index + 1, category: category, widgetState: widgetState)
            }
            Spacer()

            // Open Web App button
            Button(action: {
                openWebApp()
            }) {
                HStack(spacing: 6) {
                    Image(systemName: "safari")
                        .font(.system(size: Typography.labelBold))
                    Text("Open Web")
                        .font(.system(size: Typography.labelBold))
                }
                .foregroundColor(StyleTokens.secondaryText)
                .frame(maxWidth: .infinity)
                .padding(.vertical, 8)
            }
            .buttonStyle(PlainButtonStyle())
            .background(StyleTokens.baseVoid)
            .overlay(
                Rectangle()
                    .fill(StyleTokens.structuralBorder.opacity(Palette.subtleLineOpacity))
                    .frame(height: 1),
                alignment: .top
            )

            // Logout button at the bottom
            Button(action: {
                logout()
            }) {
                HStack(spacing: 6) {
                    Image(systemName: "rectangle.portrait.and.arrow.right")
                        .font(.system(size: Typography.labelBold))
                    Text("Logout")
                        .font(.system(size: Typography.labelBold))
                }
                .foregroundColor(StyleTokens.secondaryText)
                .frame(maxWidth: .infinity)
                .padding(.vertical, 8)
            }
            .buttonStyle(PlainButtonStyle())
            .background(StyleTokens.baseVoid)
            .overlay(
                Rectangle()
                    .fill(StyleTokens.structuralBorder.opacity(Palette.subtleLineOpacity))
                    .frame(height: 1),
                alignment: .top
            )
        }
        .background(StyleTokens.baseVoid)
    }

    private func openWebApp() {
        print("[WEB] Opening web app in browser...")
        if let url = URL(string: BuildConfig.webAppBaseURL) {
            NSWorkspace.shared.open(url)
            print("[WEB] Browser opened: \(BuildConfig.webAppBaseURL)")
        } else {
            print("[ERROR] Invalid web app URL: \(BuildConfig.webAppBaseURL)")
        }
    }

    private func logout() {
        print("[AUTH] Logging out...")
        APIClient.shared.clearAuthToken()
        widgetState.stopPeriodicRefresh()
        isAuthenticated = false
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
                    .font(.system(size: Typography.categoryRowName, weight: .bold, design: .default))
                    .foregroundColor(isActive ? .white : StyleTokens.secondaryText)
                    .tracking(1.2)
                    .lineLimit(1)
                    .truncationMode(.tail)

                Spacer(minLength: 0)

                // Index number
                Text("\(index)")
                    .font(.system(size: Typography.tinyMono, design: .monospaced))
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

    /// Calculate the relative luminance (WCAG formula)
    /// Returns value from 0.0 (darkest) to 1.0 (lightest)
    private func relativeLuminance(r: Double, g: Double, b: Double) -> Double {
        func adjust(_ channel: Double) -> Double {
            return channel <= 0.03928 ? channel / 12.92 : pow((channel + 0.055) / 1.055, 2.4)
        }
        return 0.2126 * adjust(r) + 0.7152 * adjust(g) + 0.0722 * adjust(b)
    }

    /// Returns optimal text color (black or white) for maximum contrast
    /// Uses WCAG 2.0 relative luminance formula
    static func contrastingTextColor(for hexColor: String) -> Color {
        let hex = hexColor.trimmingCharacters(in: CharacterSet.alphanumerics.inverted)
        var int: UInt64 = 0
        Scanner(string: hex).scanHexInt64(&int)

        let r, g, b: Double
        switch hex.count {
        case 3: // RGB (12-bit)
            r = Double((int >> 8) * 17) / 255.0
            g = Double((int >> 4 & 0xF) * 17) / 255.0
            b = Double((int & 0xF) * 17) / 255.0
        case 6: // RGB (24-bit)
            r = Double(int >> 16) / 255.0
            g = Double(int >> 8 & 0xFF) / 255.0
            b = Double(int & 0xFF) / 255.0
        case 8: // ARGB (32-bit)
            r = Double(int >> 16 & 0xFF) / 255.0
            g = Double(int >> 8 & 0xFF) / 255.0
            b = Double(int & 0xFF) / 255.0
        default:
            return .white
        }

        // Calculate relative luminance
        func adjust(_ channel: Double) -> Double {
            return channel <= 0.03928 ? channel / 12.92 : pow((channel + 0.055) / 1.055, 2.4)
        }
        let luminance = 0.2126 * adjust(r) + 0.7152 * adjust(g) + 0.0722 * adjust(b)

        // Use white text for dark backgrounds (luminance < 0.5), black for light
        return luminance < 0.5 ? .white : .black
    }
}

extension View {
    func fontVariantMonospacedDigit() -> some View {
        self.monospacedDigit()
    }
}
