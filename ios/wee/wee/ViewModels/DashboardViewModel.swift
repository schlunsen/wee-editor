//
//  DashboardViewModel.swift
//  wee
//
//  ViewModel for dashboard data
//

import SwiftUI
import Combine

@MainActor
class DashboardViewModel: ObservableObject {
    @Published var isLoading = false
    @Published var errorMessage: String?
    @Published var projects: [Project] = []
    @Published var recentSessions: [AgentSession] = []
    @Published var activeSessions = 0
    @Published var totalSessions = 0
    @Published var totalCost: Double = 0

    private let apiClient = WeeAPIClient.shared

    /// Recent projects sorted by most recently updated
    var recentProjects: [Project] {
        projects
            .sorted { $0.updated_at > $1.updated_at }
            .prefix(5)
            .map { $0 }
    }

    func loadData() async {
        await refresh()
    }

    func refresh() async {
        isLoading = true
        errorMessage = nil

        defer { isLoading = false }

        do {
            // Fetch projects and sessions in parallel
            async let projectsFetch = apiClient.getProjects()
            async let sessionsFetch = apiClient.getAgentSessions()

            let (fetchedProjects, fetchedSessions) = try await (projectsFetch, sessionsFetch)

            projects = fetchedProjects

            // Sort sessions by most recent and take top 8
            recentSessions = fetchedSessions
                .sorted { $0.created_at > $1.created_at }
                .prefix(8)
                .map { $0 }

            totalSessions = fetchedSessions.count

            activeSessions = fetchedSessions.filter {
                $0.status.lowercased() == "processing" || $0.status.lowercased() == "running" || $0.status.lowercased() == "active"
            }.count

            totalCost = fetchedSessions.compactMap { $0.cost }.reduce(0, +)
        } catch {
            errorMessage = error.localizedDescription
        }
    }
}
