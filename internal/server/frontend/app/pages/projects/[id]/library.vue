<template>
  <div class="library-page">
    <!-- Loading -->
    <div v-if="projectLoading" class="loading-container">
      <div class="loading-spinner"></div>
      <p>Loading library...</p>
    </div>

    <!-- Project load error -->
    <div v-else-if="projectError" class="error-container">
      <p class="error-message">{{ projectError }}</p>
      <button @click="$router.back()" class="btn-back">Back to Project</button>
    </div>

    <!-- Main content -->
    <div v-else-if="project" class="library-content">
      <!-- Header -->
      <div class="library-header">
        <div class="header-top">
          <button @click="navigateBack" class="btn-back-header">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="15 18 9 12 15 6"></polyline>
            </svg>
          </button>
          <div class="header-info">
            <span class="header-project-name">{{ project.name }}</span>
            <h1>Library</h1>
          </div>
        </div>

        <!-- Tab bar -->
        <div class="tab-bar">
          <button
            v-for="tab in tabs"
            :key="tab.id"
            class="tab-btn"
            :class="{ active: activeTab === tab.id }"
            @click="switchTab(tab.id)"
          >
            {{ tab.label }}
            <span v-if="tab.count !== undefined" class="tab-count">{{ tab.count }}</span>
          </button>
        </div>

        <!-- Search -->
        <div class="search-bar">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="11" cy="11" r="8"></circle>
            <line x1="21" y1="21" x2="16.65" y2="16.65"></line>
          </svg>
          <input
            v-model="searchQuery"
            type="text"
            :placeholder="`Search ${activeTabLabel}...`"
            class="search-input"
          />
          <button v-if="searchQuery" @click="searchQuery = ''" class="search-clear">Clear</button>
        </div>
      </div>

      <!-- Success banner -->
      <Transition name="fade">
        <div v-if="successMessage" class="success-banner">
          <span>{{ successMessage }}</span>
          <button @click="successMessage = null" class="banner-dismiss">Dismiss</button>
        </div>
      </Transition>

      <!-- Error banner -->
      <Transition name="fade">
        <div v-if="errorMessage" class="error-banner">
          <span>{{ errorMessage }}</span>
          <button @click="errorMessage = null" class="banner-dismiss">Dismiss</button>
        </div>
      </Transition>

      <!-- Tab: Skills -->
      <div v-if="activeTab === 'skills'" class="tab-content">
        <div class="tab-actions">
          <button @click="discoverSkills" class="btn-action" :disabled="skillsDiscovering">
            <div v-if="skillsDiscovering" class="btn-spinner-small"></div>
            <svg v-else width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="11" cy="11" r="8"></circle>
              <line x1="21" y1="21" x2="16.65" y2="16.65"></line>
            </svg>
            {{ skillsDiscovering ? 'Discovering...' : 'Discover Skills' }}
          </button>
          <button @click="showTemplates = !showTemplates" class="btn-action btn-templates">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <rect x="3" y="3" width="7" height="7"></rect>
              <rect x="14" y="3" width="7" height="7"></rect>
              <rect x="14" y="14" width="7" height="7"></rect>
              <rect x="3" y="14" width="7" height="7"></rect>
            </svg>
            {{ showTemplates ? 'Hide Templates' : 'Browse Templates' }}
          </button>
        </div>

        <!-- Skill Templates Gallery -->
        <Transition name="fade">
          <div v-if="showTemplates" class="templates-section">
            <div class="templates-header">
              <h2 class="section-title">Skill Templates</h2>
              <span class="templates-count">{{ filteredTemplates.length }} templates</span>
            </div>

            <!-- Category filter tabs -->
            <div class="category-tabs">
              <button
                class="category-tab"
                :class="{ active: selectedCategory === 'all' }"
                @click="selectedCategory = 'all'"
              >
                All
              </button>
              <button
                v-for="cat in templateCategories"
                :key="cat.name"
                class="category-tab"
                :class="{ active: selectedCategory === cat.name }"
                @click="selectedCategory = cat.name"
              >
                {{ cat.emoji }} {{ cat.name }}
                <span class="cat-count">{{ cat.count }}</span>
              </button>
            </div>

            <div v-if="templatesLoading" class="tab-loading">
              <div class="loading-spinner"></div>
              <p>Loading templates...</p>
            </div>

            <div v-else-if="filteredTemplates.length === 0" class="empty-state-inline">
              <p>No templates found.</p>
            </div>

            <div v-else class="card-grid">
              <div
                v-for="tpl in filteredTemplates"
                :key="tpl.name"
                class="resource-card template-card"
                @click="installTemplate(tpl)"
              >
                <div class="card-header">
                  <code class="skill-name">/{{ tpl.name }}</code>
                  <span class="badge badge-category">{{ tpl.category }}</span>
                </div>
                <p class="card-description">{{ tpl.description }}</p>
                <div v-if="tpl.frontmatter?.['argument-hint']" class="card-meta">
                  <span class="meta-label">Args:</span>
                  <code class="meta-value">{{ tpl.frontmatter['argument-hint'] }}</code>
                </div>
                <div class="template-install-hint">
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <line x1="12" y1="5" x2="12" y2="19"></line>
                    <line x1="5" y1="12" x2="19" y2="12"></line>
                  </svg>
                  Click to add
                </div>
              </div>
            </div>
          </div>
        </Transition>

        <!-- Installed Skills -->
        <div class="installed-skills-section">
          <h2 v-if="showTemplates && filteredSkills.length > 0" class="section-title">Installed Skills</h2>

          <div v-if="skillsLoading" class="tab-loading">
            <div class="loading-spinner"></div>
            <p>Loading skills...</p>
          </div>

          <div v-else-if="filteredSkills.length === 0 && !showTemplates" class="empty-state">
            <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
              <polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"></polygon>
            </svg>
            <p v-if="searchQuery">No skills matching "{{ searchQuery }}"</p>
            <p v-else>No skills found. Try "Browse Templates" to add skills, or "Discover Skills" to scan the filesystem.</p>
          </div>

          <div v-else-if="filteredSkills.length === 0 && showTemplates" class="empty-state-inline">
            <p v-if="searchQuery">No installed skills matching "{{ searchQuery }}"</p>
            <p v-else>No skills installed yet. Click a template above to add one.</p>
          </div>

          <div v-else class="card-grid">
            <div v-for="skill in filteredSkills" :key="skill.name" class="resource-card" :class="{ 'is-default-skill': isDefaultSkill(skill.name) }">
              <div class="card-header">
                <code class="skill-name">/{{ skill.name }}</code>
                <div class="badge-row">
                  <span class="badge" :class="scopeBadgeClass(skill.scope)">{{ skill.scope }}</span>
                  <span v-if="skill.effort" class="badge badge-effort">{{ skill.effort }}</span>
                  <span v-if="isDefaultSkill(skill.name)" class="badge badge-default">AUTO</span>
                </div>
              </div>
              <p class="card-description">{{ skill.description || 'No description' }}</p>
              <div class="card-footer">
                <div v-if="skill.source_file" class="card-meta">
                  <span class="meta-label">Source:</span>
                  <span class="meta-value">{{ truncate(skill.source_file, 60) }}</span>
                </div>
                <button
                  class="btn-default-toggle"
                  :class="{ active: isDefaultSkill(skill.name) }"
                  @click="toggleDefaultSkill(skill.name)"
                  :disabled="togglingSkill === skill.name"
                  :title="isDefaultSkill(skill.name) ? 'Remove from auto-enabled skills' : 'Auto-enable for new sessions'"
                >
                  <svg v-if="togglingSkill === skill.name" class="btn-spinner-tiny" width="14" height="14" viewBox="0 0 24 24">
                    <circle cx="12" cy="12" r="10" fill="none" stroke="currentColor" stroke-width="2" stroke-dasharray="31.4" stroke-dashoffset="10" />
                  </svg>
                  <template v-else>
                    {{ isDefaultSkill(skill.name) ? '✓ Auto-enabled' : 'Auto-enable' }}
                  </template>
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Tab: Hooks -->
      <div v-if="activeTab === 'hooks'" class="tab-content">
        <div v-if="hooksLoading" class="tab-loading">
          <div class="loading-spinner"></div>
          <p>Loading hooks...</p>
        </div>

        <div v-else-if="filteredHookEvents.length === 0" class="empty-state">
          <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <path d="M13 2L3 14h9l-1 8 10-12h-9l1-8z"></path>
          </svg>
          <p v-if="searchQuery">No hooks matching "{{ searchQuery }}"</p>
          <p v-else>No hooks configured for this project.</p>
        </div>

        <div v-else class="hooks-list">
          <div
            v-for="event in filteredHookEvents"
            :key="event.name"
            class="hook-event-group"
          >
            <button class="hook-event-header" @click="toggleEventGroup(event.name)">
              <svg
                width="16" height="16" viewBox="0 0 24 24" fill="none"
                stroke="currentColor" stroke-width="2"
                class="chevron"
                :class="{ rotated: expandedEvents.has(event.name) }"
              >
                <polyline points="9 18 15 12 9 6"></polyline>
              </svg>
              <span class="event-name">{{ event.name }}</span>
              <span class="event-count">{{ event.hooks.length }}</span>
            </button>

            <Transition name="collapse">
              <div v-if="expandedEvents.has(event.name)" class="hook-event-body">
                <div v-for="(hook, idx) in event.hooks" :key="idx" class="hook-item">
                  <div class="hook-badges">
                    <span class="badge" :class="hookTypeBadgeClass(hook.type)">{{ hook.type }}</span>
                    <span class="badge" :class="hookSourceBadgeClass(hook.source)">{{ hook.source }}</span>
                  </div>
                  <div class="hook-detail">
                    <template v-if="hook.type === 'command'">
                      <code class="hook-command">{{ truncate(hook.command || '', 100) }}</code>
                    </template>
                    <template v-else-if="hook.type === 'prompt'">
                      <span class="hook-prompt">{{ truncate(hook.prompt || '', 120) }}</span>
                    </template>
                    <template v-else-if="hook.type === 'http'">
                      <code class="hook-command">{{ hook.method || 'POST' }} {{ truncate(hook.url || '', 80) }}</code>
                    </template>
                    <template v-else>
                      <span class="hook-prompt">{{ truncate(hook.description || hook.type || '', 120) }}</span>
                    </template>
                  </div>
                </div>
              </div>
            </Transition>
          </div>
        </div>
      </div>

      <!-- Tab: MCP Servers -->
      <div v-if="activeTab === 'mcps'" class="tab-content">
        <div class="tab-actions">
          <button @click="showAddMcpForm = !showAddMcpForm" class="btn-action">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <line x1="12" y1="5" x2="12" y2="19"></line>
              <line x1="5" y1="12" x2="19" y2="12"></line>
            </svg>
            {{ showAddMcpForm ? 'Cancel' : 'Add Custom' }}
          </button>
        </div>

        <!-- Add Custom MCP form -->
        <Transition name="fade">
          <div v-if="showAddMcpForm" class="add-mcp-form">
            <h3>Add Custom MCP Server</h3>
            <div class="form-group">
              <label>Server Name</label>
              <input v-model="newMcp.name" type="text" placeholder="my-server" class="form-input" />
            </div>
            <div class="form-group">
              <label>Command</label>
              <input v-model="newMcp.command" type="text" placeholder="npx -y @modelcontextprotocol/server-filesystem" class="form-input" />
            </div>
            <div class="form-group">
              <label>Args (comma-separated)</label>
              <input v-model="newMcp.argsStr" type="text" placeholder="/path/to/dir, --flag" class="form-input" />
            </div>
            <div class="form-actions">
              <button @click="addCustomMcp" class="btn-primary" :disabled="mcpAdding || !newMcp.name || !newMcp.command">
                <div v-if="mcpAdding" class="btn-spinner-small"></div>
                {{ mcpAdding ? 'Adding...' : 'Add Server' }}
              </button>
              <button @click="showAddMcpForm = false" class="btn-secondary">Cancel</button>
            </div>
          </div>
        </Transition>

        <div v-if="mcpsLoading" class="tab-loading">
          <div class="loading-spinner"></div>
          <p>Loading MCP servers...</p>
        </div>

        <template v-else>
          <!-- Installed servers -->
          <div class="mcp-section">
            <h2 class="section-title">Installed</h2>
            <div v-if="filteredInstalledMcps.length === 0" class="empty-state-inline">
              <p>No MCP servers installed in this project.</p>
            </div>
            <div v-else class="card-grid">
              <div v-for="(server, name) in filteredInstalledMcps" :key="name" class="resource-card mcp-card">
                <div class="card-header">
                  <span class="mcp-name">{{ name }}</span>
                  <span class="badge badge-installed">installed</span>
                </div>
                <div class="card-description">
                  <code v-if="server.command" class="mcp-command">{{ server.command }} {{ (server.args || []).join(' ') }}</code>
                  <code v-else-if="server.url" class="mcp-command">{{ server.url }}</code>
                </div>
                <div class="card-actions">
                  <button
                    @click="removeMcpServer(name as string)"
                    class="btn-danger-small"
                    :disabled="mcpRemoving === name"
                  >
                    {{ mcpRemoving === name ? 'Removing...' : 'Remove' }}
                  </button>
                </div>
              </div>
            </div>
          </div>

        </template>
      </div>

      <!-- Tab: Packs -->
      <div v-if="activeTab === 'packs'" class="tab-content">
        <div v-if="packsLoading" class="tab-loading">
          <div class="loading-spinner"></div>
          <p>Loading packs...</p>
        </div>

        <div v-else-if="filteredPacks.length === 0" class="empty-state">
          <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"></path>
            <polyline points="3.27 6.96 12 12.01 20.73 6.96"></polyline>
            <line x1="12" y1="22.08" x2="12" y2="12"></line>
          </svg>
          <p v-if="searchQuery">No packs matching "{{ searchQuery }}"</p>
          <p v-else>No packs available.</p>
        </div>

        <div v-else class="card-grid">
          <div v-for="pack in filteredPacks" :key="pack.name" class="resource-card pack-card">
            <div class="card-header">
              <div class="pack-title-row">
                <span v-if="pack.icon" class="pack-icon">{{ pack.icon }}</span>
                <span class="pack-name">{{ pack.name }}</span>
              </div>
              <div class="badge-row">
                <span v-if="pack.category" class="badge badge-category">{{ pack.category }}</span>
                <span v-if="pack.installed" class="badge badge-installed">installed</span>
              </div>
            </div>
            <p class="card-description">{{ pack.description || 'No description' }}</p>

            <div class="pack-stats">
              <span v-if="pack.version" class="pack-stat">v{{ pack.version }}</span>
              <span v-if="pack.skill_count != null" class="pack-stat">{{ pack.skill_count }} skills</span>
              <span v-if="pack.hook_count != null" class="pack-stat">{{ pack.hook_count }} hooks</span>
            </div>

            <!-- Expandable details -->
            <button
              v-if="pack.skills?.length || pack.hooks?.length"
              class="pack-details-toggle"
              @click="togglePackDetails(pack.name)"
            >
              {{ expandedPacks.has(pack.name) ? 'Hide details' : 'Show details' }}
            </button>

            <Transition name="collapse">
              <div v-if="expandedPacks.has(pack.name)" class="pack-details">
                <div v-if="pack.skills?.length" class="pack-detail-section">
                  <h4>Skills</h4>
                  <ul>
                    <li v-for="s in pack.skills" :key="s.name || s">
                      <code>/{{ typeof s === 'string' ? s : s.name }}</code>
                    </li>
                  </ul>
                </div>
                <div v-if="pack.hooks?.length" class="pack-detail-section">
                  <h4>Hooks</h4>
                  <ul>
                    <li v-for="h in pack.hooks" :key="h.event || h">
                      {{ typeof h === 'string' ? h : `${h.event} (${h.type})` }}
                    </li>
                  </ul>
                </div>
              </div>
            </Transition>

            <div class="card-actions">
              <button
                v-if="pack.installed"
                @click="uninstallPack(pack)"
                class="btn-danger-small"
                :disabled="packActioning === pack.name"
              >
                {{ packActioning === pack.name ? 'Uninstalling...' : 'Uninstall' }}
              </button>
              <button
                v-else
                @click="installPack(pack)"
                class="btn-primary-small"
                :disabled="packActioning === pack.name"
              >
                {{ packActioning === pack.name ? 'Installing...' : 'Install' }}
              </button>
            </div>
          </div>
        </div>
      </div>
      <!-- Tab: Design Styles (Refero) -->
      <div v-if="activeTab === 'design'" class="tab-content">
        <!-- Scheme filter -->
        <div class="tab-actions">
          <div class="scheme-filters">
            <button
              class="category-tab"
              :class="{ active: referoSchemeFilter === 'all' }"
              @click="referoSchemeFilter = 'all'"
            >All</button>
            <button
              class="category-tab"
              :class="{ active: referoSchemeFilter === 'light' }"
              @click="referoSchemeFilter = 'light'"
            >☀️ Light</button>
            <button
              class="category-tab"
              :class="{ active: referoSchemeFilter === 'dark' }"
              @click="referoSchemeFilter = 'dark'"
            >🌙 Dark</button>
          </div>
        </div>

        <div v-if="referoLoading" class="tab-loading">
          <div class="loading-spinner"></div>
          <p>Loading design styles from Refero...</p>
        </div>

        <div v-else-if="filteredReferoStyles.length === 0" class="empty-state">
          <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <circle cx="12" cy="12" r="10"></circle>
            <path d="M8 14s1.5 2 4 2 4-2 4-2"></path>
            <line x1="9" y1="9" x2="9.01" y2="9"></line>
            <line x1="15" y1="9" x2="15.01" y2="9"></line>
          </svg>
          <p v-if="searchQuery">No design styles matching "{{ searchQuery }}"</p>
          <p v-else>No design styles available. Check your connection.</p>
        </div>

        <template v-else>
          <div class="card-grid refero-grid">
            <div
              v-for="style in filteredReferoStyles"
              :key="style.id"
              class="resource-card refero-card"
              @click="referoPreview = style"
            >
              <!-- Thumbnail -->
              <div class="refero-thumbnail">
                <img
                  v-if="style.thumbnailUrl"
                  :src="style.thumbnailUrl"
                  :alt="style.siteName"
                  loading="lazy"
                  @error="($event.target as HTMLImageElement).style.display = 'none'"
                />
                <div v-else class="refero-thumbnail-placeholder">
                  <span>{{ style.siteName.charAt(0) }}</span>
                </div>
              </div>

              <!-- Info -->
              <div class="refero-info">
                <div class="card-header">
                  <span class="refero-site-name">{{ style.siteName }}</span>
                  <span class="badge" :class="style.colorScheme === 'dark' ? 'badge-dark-scheme' : 'badge-light-scheme'">
                    {{ style.colorScheme }}
                  </span>
                </div>

                <p v-if="style.northStar" class="card-description refero-north-star">{{ style.northStar }}</p>

                <!-- Color swatches -->
                <div v-if="style.colors?.length" class="refero-swatches">
                  <div
                    v-for="(color, idx) in style.colors.slice(0, 6)"
                    :key="idx"
                    class="refero-swatch"
                    :style="{ backgroundColor: color.hex }"
                    :title="color.name ? `${color.name}: ${color.hex}` : color.hex"
                  ></div>
                  <span v-if="style.colors.length > 6" class="refero-swatch-more">+{{ style.colors.length - 6 }}</span>
                </div>

                <!-- Fonts -->
                <div v-if="style.fonts?.length" class="refero-fonts">
                  <span v-for="font in style.fonts.slice(0, 2)" :key="font" class="refero-font-tag">{{ font }}</span>
                </div>
              </div>
            </div>
          </div>

          <!-- Load more -->
          <div v-if="referoHasMore" class="refero-load-more">
            <button
              @click="loadMoreReferoStyles"
              class="btn-action"
              :disabled="referoLoadingMore"
            >
              <div v-if="referoLoadingMore" class="btn-spinner-small"></div>
              {{ referoLoadingMore ? 'Loading...' : 'Load More Styles' }}
            </button>
          </div>
        </template>

        <!-- Style Preview / Import Modal -->
        <Transition name="fade">
          <div v-if="referoPreview" class="refero-modal-backdrop" @click.self="referoPreview = null">
            <div class="refero-modal">
              <div class="refero-modal-header">
                <div class="refero-modal-title-row">
                  <img
                    v-if="referoPreview.iconUrl"
                    :src="referoPreview.iconUrl"
                    class="refero-modal-icon"
                    @error="($event.target as HTMLImageElement).style.display = 'none'"
                  />
                  <div>
                    <h2>{{ referoPreview.siteName }}</h2>
                    <a :href="referoPreview.url" target="_blank" rel="noopener" class="refero-modal-url">{{ referoPreview.url }}</a>
                  </div>
                </div>
                <button @click="referoPreview = null" class="refero-modal-close">&times;</button>
              </div>

              <div class="refero-modal-body">
                <!-- Screenshot -->
                <div v-if="referoPreview.screenshotUrl" class="refero-modal-screenshot">
                  <img
                    :src="referoPreview.screenshotUrl"
                    :alt="referoPreview.siteName"
                    loading="lazy"
                  />
                </div>

                <!-- North Star -->
                <p v-if="referoPreview.northStar" class="refero-modal-north-star">
                  "{{ referoPreview.northStar }}"
                </p>

                <!-- Metadata -->
                <div class="refero-modal-meta">
                  <div class="refero-modal-meta-item">
                    <span class="meta-label">Scheme</span>
                    <span class="badge" :class="referoPreview.colorScheme === 'dark' ? 'badge-dark-scheme' : 'badge-light-scheme'">
                      {{ referoPreview.colorScheme }}
                    </span>
                  </div>
                </div>

                <!-- Colors -->
                <div v-if="referoPreview.colors?.length" class="refero-modal-section">
                  <h3>Colors</h3>
                  <div class="refero-modal-colors">
                    <div
                      v-for="(color, idx) in referoPreview.colors"
                      :key="idx"
                      class="refero-modal-color"
                    >
                      <div class="refero-modal-color-swatch" :style="{ backgroundColor: color.hex }"></div>
                      <div class="refero-modal-color-info">
                        <code>{{ color.hex }}</code>
                        <span v-if="color.name" class="refero-modal-color-name">{{ color.name }}</span>
                      </div>
                    </div>
                  </div>
                </div>

                <!-- Fonts -->
                <div v-if="referoPreview.fonts?.length" class="refero-modal-section">
                  <h3>Typography</h3>
                  <div class="refero-modal-fonts">
                    <span v-for="font in referoPreview.fonts" :key="font" class="refero-font-tag-large">{{ font }}</span>
                  </div>
                </div>
              </div>

              <div class="refero-modal-footer">
                <button @click="referoPreview = null" class="btn-secondary">Cancel</button>
                <button
                  @click="importReferoStyle(referoPreview!)"
                  class="btn-primary"
                  :disabled="referoImporting === referoPreview?.id"
                >
                  <div v-if="referoImporting === referoPreview?.id" class="btn-spinner-small" style="border-top-color: #fff;"></div>
                  {{ referoImporting === referoPreview?.id ? 'Importing...' : 'Import as DESIGN.md' }}
                </button>
              </div>
            </div>
          </div>
        </Transition>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'

// ---------- Types ----------

interface Project {
  id: string
  name: string
  path: string
  description?: string
  [key: string]: unknown
}

interface Skill {
  name: string
  description?: string
  scope?: string
  effort?: string
  source_file?: string
  [key: string]: unknown
}

interface Hook {
  type: string
  source: string
  command?: string
  prompt?: string
  url?: string
  method?: string
  description?: string
  [key: string]: unknown
}

interface HookEventGroup {
  name: string
  hooks: Hook[]
}

interface McpServer {
  command?: string
  args?: string[]
  url?: string
  env?: Record<string, string>
  [key: string]: unknown
}

interface Pack {
  name: string
  description?: string
  icon?: string
  version?: string
  category?: string
  installed: boolean
  skill_count?: number
  hook_count?: number
  skills?: Array<{ name: string } | string>
  hooks?: Array<{ event: string; type: string } | string>
  [key: string]: unknown
}

interface ReferoStyle {
  id: string
  url: string
  siteName: string
  screenshotUrl: string
  thumbnailUrl: string
  iconUrl: string
  colorScheme: string
  colors: Array<{ name: string; hex: string }>
  fonts: string[]
  northStar: string
  createdAt: string
}

type TabId = 'skills' | 'hooks' | 'mcps' | 'packs' | 'design'

// ---------- Composables ----------

const route = useRoute()
const router = useRouter()
const { fetchWithAuth } = useAuthenticatedFetch()

// ---------- Core state ----------

const projectId = computed(() => route.params.id as string)
const project = ref<Project | null>(null)
const projectLoading = ref(true)
const projectError = ref<string | null>(null)

const activeTab = ref<TabId>('skills')
const searchQuery = ref('')

const successMessage = ref<string | null>(null)
const errorMessage = ref<string | null>(null)

let successTimer: ReturnType<typeof setTimeout> | null = null

// ---------- Skills state ----------

const skills = ref<Skill[]>([])
const skillsLoading = ref(false)
const skillsDiscovering = ref(false)

// ---------- Default skills state ----------

const defaultSkillIDs = ref<string[]>([])
const togglingSkill = ref<string | null>(null)

// ---------- Templates state ----------

interface SkillTemplate {
  name: string
  description: string
  category: string
  frontmatter: Record<string, any>
  body: string
}

const templates = ref<SkillTemplate[]>([])
const templatesLoading = ref(false)
const showTemplates = ref(false)
const selectedCategory = ref('all')
const templateInstalling = ref<string | null>(null)

// ---------- Hooks state ----------

const hooks = ref<Hook[]>([])
const hooksByEvent = ref<Record<string, Hook[]>>({})
const hooksLoading = ref(false)
const expandedEvents = ref<Set<string>>(new Set())

// ---------- MCP state ----------

const installedMcpServers = ref<Record<string, McpServer>>({})
const mcpsLoading = ref(false)
const mcpRemoving = ref<string | null>(null)
const mcpInstalling = ref<string | null>(null)
const mcpAdding = ref(false)
const showAddMcpForm = ref(false)
const newMcp = ref({ name: '', command: '', argsStr: '' })

// ---------- Packs state ----------

const packs = ref<Pack[]>([])
const packsLoading = ref(false)
const packActioning = ref<string | null>(null)
const expandedPacks = ref<Set<string>>(new Set())

// ---------- Design Styles (Refero) state ----------

const referoStyles = ref<ReferoStyle[]>([])
const referoLoading = ref(false)
const referoPage = ref(1)
const referoHasMore = ref(true)
const referoLoadingMore = ref(false)
const referoImporting = ref<string | null>(null)
const referoPreview = ref<ReferoStyle | null>(null)
const referoSchemeFilter = ref<string>('all')

// ---------- Tab definitions ----------

const tabs = computed(() => [
  { id: 'skills' as TabId, label: 'Skills', count: skills.value.length },
  { id: 'hooks' as TabId, label: 'Hooks', count: hooks.value.length },
  { id: 'mcps' as TabId, label: 'MCP Servers', count: Object.keys(installedMcpServers.value).length },
  { id: 'packs' as TabId, label: 'Packs', count: packs.value.length },
  { id: 'design' as TabId, label: 'Design Styles', count: referoStyles.value.length },
])

const activeTabLabel = computed(() => {
  const tab = tabs.value.find(t => t.id === activeTab.value)
  return tab?.label || ''
})

// ---------- Filtered data ----------

const filteredSkills = computed(() => {
  if (!searchQuery.value) return skills.value
  const q = searchQuery.value.toLowerCase()
  return skills.value.filter(s =>
    s.name.toLowerCase().includes(q) ||
    (s.description || '').toLowerCase().includes(q) ||
    (s.scope || '').toLowerCase().includes(q)
  )
})

// Template category metadata
const categoryEmojis: Record<string, string> = {
  devops: '🚀',
  review: '👀',
  research: '🔍',
  testing: '🧪',
  refactoring: '♻️',
  documentation: '📝',
  security: '🔒',
  debugging: '🐛',
  architecture: '🏗️',
  git: '📦',
  quality: '✨',
  general: '⚡',
}

const templateCategories = computed(() => {
  const counts: Record<string, number> = {}
  for (const t of templates.value) {
    counts[t.category] = (counts[t.category] || 0) + 1
  }
  return Object.entries(counts)
    .map(([name, count]) => ({ name, count, emoji: categoryEmojis[name] || '⚡' }))
    .sort((a, b) => b.count - a.count)
})

const filteredTemplates = computed(() => {
  let list = templates.value
  if (selectedCategory.value !== 'all') {
    list = list.filter(t => t.category === selectedCategory.value)
  }
  if (searchQuery.value) {
    const q = searchQuery.value.toLowerCase()
    list = list.filter(t =>
      t.name.toLowerCase().includes(q) ||
      t.description.toLowerCase().includes(q) ||
      t.category.toLowerCase().includes(q)
    )
  }
  // Hide templates that are already installed
  const installedNames = new Set(skills.value.map(s => s.name))
  return list.filter(t => !installedNames.has(t.name))
})

const hookEvents = computed<HookEventGroup[]>(() => {
  const events: HookEventGroup[] = []
  for (const [eventName, eventHooks] of Object.entries(hooksByEvent.value)) {
    events.push({ name: eventName, hooks: eventHooks })
  }
  return events.sort((a, b) => a.name.localeCompare(b.name))
})

const filteredHookEvents = computed(() => {
  if (!searchQuery.value) return hookEvents.value
  const q = searchQuery.value.toLowerCase()
  return hookEvents.value
    .map(event => ({
      ...event,
      hooks: event.hooks.filter(h =>
        event.name.toLowerCase().includes(q) ||
        (h.command || '').toLowerCase().includes(q) ||
        (h.prompt || '').toLowerCase().includes(q) ||
        h.type.toLowerCase().includes(q) ||
        h.source.toLowerCase().includes(q)
      )
    }))
    .filter(event => event.hooks.length > 0)
})

const filteredInstalledMcps = computed(() => {
  if (!searchQuery.value) return installedMcpServers.value
  const q = searchQuery.value.toLowerCase()
  const result: Record<string, McpServer> = {}
  for (const [name, server] of Object.entries(installedMcpServers.value)) {
    if (
      name.toLowerCase().includes(q) ||
      (server.command || '').toLowerCase().includes(q) ||
      (server.url || '').toLowerCase().includes(q)
    ) {
      result[name] = server
    }
  }
  return result
})

const filteredPacks = computed(() => {
  if (!searchQuery.value) return packs.value
  const q = searchQuery.value.toLowerCase()
  return packs.value.filter(p =>
    p.name.toLowerCase().includes(q) ||
    (p.description || '').toLowerCase().includes(q) ||
    (p.category || '').toLowerCase().includes(q)
  )
})

const filteredReferoStyles = computed(() => {
  let list = referoStyles.value
  if (referoSchemeFilter.value !== 'all') {
    list = list.filter(s => s.colorScheme === referoSchemeFilter.value)
  }
  if (searchQuery.value) {
    const q = searchQuery.value.toLowerCase()
    list = list.filter(s =>
      s.siteName.toLowerCase().includes(q) ||
      s.url.toLowerCase().includes(q) ||
      (s.northStar || '').toLowerCase().includes(q) ||
      s.fonts.some(f => f.toLowerCase().includes(q))
    )
  }
  return list
})

// ---------- Helpers ----------

function truncate(text: string, max: number): string {
  if (!text || text.length <= max) return text
  return text.substring(0, max) + '...'
}

function scopeBadgeClass(scope?: string): string {
  if (scope === 'project') return 'badge-project'
  if (scope === 'personal') return 'badge-personal'
  return 'badge-default'
}

function hookTypeBadgeClass(type: string): string {
  const map: Record<string, string> = {
    command: 'badge-command',
    prompt: 'badge-prompt',
    http: 'badge-http',
    agent: 'badge-agent',
  }
  return map[type] || 'badge-default'
}

function hookSourceBadgeClass(source: string): string {
  const map: Record<string, string> = {
    global: 'badge-global',
    project: 'badge-project',
    local: 'badge-local',
  }
  return map[source] || 'badge-default'
}

function isServerInstalled(name: string): boolean {
  return name in installedMcpServers.value
}

function showSuccess(msg: string) {
  successMessage.value = msg
  if (successTimer) clearTimeout(successTimer)
  successTimer = setTimeout(() => {
    successMessage.value = null
  }, 3000)
}

function showError(msg: string) {
  errorMessage.value = msg
}

// ---------- Navigation ----------

function navigateBack() {
  router.push(`/projects/${projectId.value}`)
}

function switchTab(tab: TabId) {
  activeTab.value = tab
  searchQuery.value = ''
  router.replace({ query: { ...route.query, tab } })
}

function toggleEventGroup(name: string) {
  const s = new Set(expandedEvents.value)
  if (s.has(name)) {
    s.delete(name)
  } else {
    s.add(name)
  }
  expandedEvents.value = s
}

function togglePackDetails(name: string) {
  const s = new Set(expandedPacks.value)
  if (s.has(name)) {
    s.delete(name)
  } else {
    s.add(name)
  }
  expandedPacks.value = s
}

// ---------- Data loaders ----------

async function loadProject() {
  try {
    projectLoading.value = true
    projectError.value = null
    const res = await fetchWithAuth(`/api/projects/${projectId.value}`)
    if (!res.ok) throw new Error('Failed to load project')
    const data = await res.json()
    project.value = data.project || data
  } catch (err: any) {
    projectError.value = err.message || 'Failed to load project'
  } finally {
    projectLoading.value = false
  }
}

async function loadSkills() {
  try {
    skillsLoading.value = true
    const res = await fetchWithAuth('/api/skills/')
    if (!res.ok) throw new Error('Failed to load skills')
    const data = await res.json()
    skills.value = data.skills || []

    // Auto-show templates gallery when the user has few installed skills
    // so new users immediately see what's available
    if (skills.value.length <= 3 && activeTab.value === 'skills') {
      showTemplates.value = true
    }
  } catch (err: any) {
    console.error('Failed to load skills:', err)
  } finally {
    skillsLoading.value = false
  }
}

async function loadDefaultSkills() {
  if (!projectId.value) return
  try {
    const res = await fetchWithAuth(`/api/projects/${projectId.value}/default-skills`)
    if (!res.ok) return
    const data = await res.json()
    defaultSkillIDs.value = data.skill_ids || []
  } catch (err: any) {
    console.error('Failed to load default skills:', err)
  }
}

function isDefaultSkill(skillName: string): boolean {
  return defaultSkillIDs.value.includes(skillName)
}

async function toggleDefaultSkill(skillName: string) {
  if (!projectId.value) return
  togglingSkill.value = skillName
  try {
    const res = await fetchWithAuth(`/api/projects/${projectId.value}/default-skills/toggle`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ skill_name: skillName }),
    })
    if (!res.ok) throw new Error('Failed to toggle default skill')
    const data = await res.json()
    defaultSkillIDs.value = data.skill_ids || []
    showSuccess(data.added ? `"/${skillName}" will auto-enable for new sessions` : `"/${skillName}" removed from auto-enabled skills`)
  } catch (err: any) {
    showError(err.message || 'Failed to toggle default skill')
  } finally {
    togglingSkill.value = null
  }
}

async function loadHooks() {
  if (!project.value) return
  try {
    hooksLoading.value = true
    const res = await fetchWithAuth(`/api/hooks/?project_dir=${encodeURIComponent(project.value.path)}`)
    if (!res.ok) throw new Error('Failed to load hooks')
    const data = await res.json()
    hooks.value = data.hooks || []
    hooksByEvent.value = data.by_event || {}
  } catch (err: any) {
    console.error('Failed to load hooks:', err)
  } finally {
    hooksLoading.value = false
  }
}

async function loadMcpServers() {
  if (!project.value) return
  try {
    mcpsLoading.value = true
    const configRes = await fetchWithAuth(`/api/projects/${projectId.value}/mcp-config`)
    if (configRes.ok) {
      const data = await configRes.json()
      installedMcpServers.value = data.mcpServers || {}
    }
  } catch (err: any) {
    console.error('Failed to load MCP servers:', err)
  } finally {
    mcpsLoading.value = false
  }
}

async function loadPacks() {
  try {
    packsLoading.value = true
    const res = await fetchWithAuth('/api/packs/')
    if (!res.ok) throw new Error('Failed to load packs')
    const data = await res.json()
    packs.value = data.packs || []
  } catch (err: any) {
    console.error('Failed to load packs:', err)
  } finally {
    packsLoading.value = false
  }
}

// ---------- Tab data loading ----------

async function loadTabData(tab: TabId) {
  switch (tab) {
    case 'skills':
      if (skills.value.length === 0) await loadSkills()
      break
    case 'hooks':
      if (hooks.value.length === 0) await loadHooks()
      break
    case 'mcps':
      if (Object.keys(installedMcpServers.value).length === 0) {
        await loadMcpServers()
      }
      break
    case 'packs':
      if (packs.value.length === 0) await loadPacks()
      break
    case 'design':
      if (referoStyles.value.length === 0) await loadReferoStyles()
      break
  }
}

// Watch tab changes to load data
watch(activeTab, (tab) => {
  loadTabData(tab)
})

// Load templates when the gallery is opened
watch(showTemplates, (show) => {
  if (show && templates.value.length === 0) {
    loadTemplates()
  }
})

// ---------- Actions ----------

async function loadTemplates() {
  try {
    templatesLoading.value = true
    const res = await fetchWithAuth('/api/skills/templates')
    if (!res.ok) throw new Error('Failed to load templates')
    const data = await res.json()
    templates.value = data.templates || []
  } catch (err: any) {
    console.error('Failed to load templates:', err)
  } finally {
    templatesLoading.value = false
  }
}

async function installTemplate(tpl: SkillTemplate) {
  try {
    templateInstalling.value = tpl.name
    const res = await fetchWithAuth('/api/skills/', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        name: tpl.name,
        description: tpl.description,
        scope: 'project',
        body: tpl.body,
        frontmatter: tpl.frontmatter,
      }),
    })
    if (!res.ok) {
      const data = await res.json()
      throw new Error(data.error || 'Failed to install skill')
    }
    showSuccess(`Skill "/${tpl.name}" added!`)
    await loadSkills()
  } catch (err: any) {
    showError(err.message || 'Failed to add skill')
  } finally {
    templateInstalling.value = null
  }
}

async function discoverSkills() {
  if (!project.value) return
  try {
    skillsDiscovering.value = true
    const res = await fetchWithAuth(
      `/api/skills/discover?project_dir=${encodeURIComponent(project.value.path)}`,
      { method: 'POST' }
    )
    if (!res.ok) throw new Error('Failed to discover skills')
    const data = await res.json()
    const count = data.discovered || data.count || 0
    showSuccess(`Discovered ${count} skill(s)`)
    await loadSkills()
  } catch (err: any) {
    showError(err.message || 'Failed to discover skills')
  } finally {
    skillsDiscovering.value = false
  }
}

async function removeMcpServer(name: string) {
  if (!confirm(`Remove MCP server "${name}"?`)) return
  try {
    mcpRemoving.value = name
    const res = await fetchWithAuth(
      `/api/projects/${projectId.value}/mcp-config/servers/${encodeURIComponent(name)}`,
      { method: 'DELETE' }
    )
    if (!res.ok) throw new Error('Failed to remove server')
    showSuccess(`Removed "${name}"`)
    await loadMcpServers()
  } catch (err: any) {
    showError(err.message || 'Failed to remove MCP server')
  } finally {
    mcpRemoving.value = null
  }
}

async function addCustomMcp() {
  if (!newMcp.value.name || !newMcp.value.command) return
  try {
    mcpAdding.value = true
    const args = newMcp.value.argsStr
      ? newMcp.value.argsStr.split(',').map(a => a.trim()).filter(Boolean)
      : []
    const res = await fetchWithAuth(
      `/api/projects/${projectId.value}/mcp-config/servers`,
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          name: newMcp.value.name,
          command: newMcp.value.command,
          args,
        }),
      }
    )
    if (!res.ok) throw new Error('Failed to add MCP server')
    showSuccess(`Added "${newMcp.value.name}"`)
    newMcp.value = { name: '', command: '', argsStr: '' }
    showAddMcpForm.value = false
    await loadMcpServers()
  } catch (err: any) {
    showError(err.message || 'Failed to add MCP server')
  } finally {
    mcpAdding.value = false
  }
}

async function installPack(pack: Pack) {
  if (!confirm(`Install pack "${pack.name}"? This will add its skills and hooks.`)) return
  try {
    packActioning.value = pack.name
    const res = await fetchWithAuth(`/api/packs/${encodeURIComponent(pack.name)}/install`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        scope: 'project',
        project_dir: project.value?.path || '',
      }),
    })
    if (!res.ok) {
      const data = await res.json().catch(() => ({}))
      throw new Error(data.error || 'Failed to install pack')
    }
    showSuccess(`Installed pack "${pack.name}"`)
    await loadPacks()
    // Refresh skills too since packs may add them
    await loadSkills()
  } catch (err: any) {
    showError(err.message || 'Failed to install pack')
  } finally {
    packActioning.value = null
  }
}

async function uninstallPack(pack: Pack) {
  if (!confirm(`Uninstall pack "${pack.name}"? Its skills and hooks will be removed.`)) return
  try {
    packActioning.value = pack.name
    const res = await fetchWithAuth(`/api/packs/${encodeURIComponent(pack.name)}`, {
      method: 'DELETE',
    })
    if (!res.ok) throw new Error('Failed to uninstall pack')
    showSuccess(`Uninstalled pack "${pack.name}"`)
    await loadPacks()
    await loadSkills()
  } catch (err: any) {
    showError(err.message || 'Failed to uninstall pack')
  } finally {
    packActioning.value = null
  }
}

// ---------- Refero actions ----------

async function loadReferoStyles(page: number = 1) {
  try {
    if (page === 1) {
      referoLoading.value = true
    } else {
      referoLoadingMore.value = true
    }
    const res = await fetchWithAuth(`/api/refero/styles?page=${page}`)
    if (!res.ok) throw new Error('Failed to fetch design styles')
    const data = await res.json()
    const styles: ReferoStyle[] = data.styles || []
    if (page === 1) {
      referoStyles.value = styles
    } else {
      referoStyles.value = [...referoStyles.value, ...styles]
    }
    referoPage.value = page
    referoHasMore.value = data.nextPage != null
  } catch (err: any) {
    console.error('Failed to load Refero styles:', err)
    if (page === 1) {
      showError('Failed to load design styles from Refero')
    }
  } finally {
    referoLoading.value = false
    referoLoadingMore.value = false
  }
}

async function loadMoreReferoStyles() {
  if (!referoHasMore.value || referoLoadingMore.value) return
  await loadReferoStyles(referoPage.value + 1)
}

async function importReferoStyle(style: ReferoStyle, overwrite: boolean = false) {
  if (!project.value) return
  try {
    referoImporting.value = style.id
    const url = overwrite ? '/api/refero/import?overwrite=true' : '/api/refero/import'
    const res = await fetchWithAuth(url, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        style_id: style.id,
        project_dir: project.value.path,
      }),
    })
    const data = await res.json()
    if (!res.ok) {
      if (data.exists) {
        // File already exists - ask to overwrite
        if (confirm(`DESIGN.md already exists in this project. Overwrite it with ${style.siteName}'s design system?`)) {
          await importReferoStyle(style, true)
          return
        }
        return
      }
      throw new Error(data.error || 'Failed to import design style')
    }
    showSuccess(data.message || `Imported ${style.siteName} design system!`)
    referoPreview.value = null
  } catch (err: any) {
    showError(err.message || 'Failed to import design style')
  } finally {
    referoImporting.value = null
  }
}

// ---------- Lifecycle ----------

onMounted(async () => {
  // Read tab from query param
  const queryTab = route.query.tab as string
  if (queryTab && ['skills', 'hooks', 'mcps', 'packs', 'design'].includes(queryTab)) {
    activeTab.value = queryTab as TabId
  }

  // Load project first
  await loadProject()

  if (project.value) {
    // Load initial tab data + preload skills for tab count + load default skills
    await Promise.all([
      loadTabData(activeTab.value),
      activeTab.value !== 'skills' ? loadSkills() : Promise.resolve(),
      activeTab.value !== 'hooks' ? loadHooks() : Promise.resolve(),
      activeTab.value !== 'packs' ? loadPacks() : Promise.resolve(),
      loadDefaultSkills(),
    ])
  }
})
</script>

<style scoped>
.library-page {
  height: 100%;
  padding: 2rem;
  max-width: 100%;
  margin: 0 auto;
  overflow-y: auto;
}

/* Loading & Error States */
.loading-container,
.error-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 400px;
  gap: 1rem;
}

.loading-spinner {
  width: 40px;
  height: 40px;
  border: 3px solid var(--border-color);
  border-top-color: var(--accent-purple);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.error-message {
  color: #dc3545;
  font-size: 1rem;
}

.btn-back {
  padding: 0.5rem 1rem;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 6px;
  color: var(--text-primary);
  cursor: pointer;
}

/* Header */
.library-header {
  margin-bottom: 1.5rem;
}

.header-top {
  display: flex;
  align-items: center;
  gap: 1rem;
  margin-bottom: 1.5rem;
}

.btn-back-header {
  padding: 0.75rem;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  color: var(--text-primary);
  cursor: pointer;
  transition: all 0.2s;
  flex-shrink: 0;
}

.btn-back-header:hover {
  background: var(--card-bg);
  border-color: var(--accent-purple);
}

.header-info {
  flex: 1;
}

.header-project-name {
  font-size: 0.85rem;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.header-info h1 {
  margin: 0.25rem 0 0 0;
  font-size: 1.75rem;
  color: var(--text-primary);
}

/* Tab bar */
.tab-bar {
  display: flex;
  gap: 0;
  border-bottom: 2px solid var(--border-color);
  margin-bottom: 1rem;
}

.tab-btn {
  padding: 0.75rem 1.25rem;
  background: none;
  border: none;
  border-bottom: 2px solid transparent;
  margin-bottom: -2px;
  color: var(--text-secondary);
  font-size: 0.95rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.tab-btn:hover {
  color: var(--text-primary);
}

.tab-btn.active {
  color: var(--accent-purple);
  border-bottom-color: var(--accent-purple);
}

.tab-count {
  background: var(--bg-secondary);
  color: var(--text-secondary);
  font-size: 0.75rem;
  padding: 0.1rem 0.5rem;
  border-radius: 10px;
  min-width: 1.5rem;
  text-align: center;
}

.tab-btn.active .tab-count {
  background: var(--accent-purple);
  color: #fff;
}

/* Search */
.search-bar {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  padding: 0.5rem 0.75rem;
  margin-bottom: 1rem;
}

.search-bar svg {
  color: var(--text-secondary);
  flex-shrink: 0;
}

.search-input {
  flex: 1;
  background: none;
  border: none;
  outline: none;
  color: var(--text-primary);
  font-size: 0.9rem;
}

.search-input::placeholder {
  color: var(--text-secondary);
}

.search-clear {
  background: none;
  border: none;
  color: var(--text-secondary);
  cursor: pointer;
  font-size: 0.8rem;
  padding: 0.2rem 0.4rem;
}

.search-clear:hover {
  color: var(--text-primary);
}

/* Banners */
.success-banner,
.error-banner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.75rem 1rem;
  border-radius: 8px;
  margin-bottom: 1rem;
  font-size: 0.9rem;
}

.success-banner {
  background: rgba(40, 167, 69, 0.15);
  border: 1px solid rgba(40, 167, 69, 0.3);
  color: #28a745;
}

.error-banner {
  background: rgba(220, 53, 69, 0.15);
  border: 1px solid rgba(220, 53, 69, 0.3);
  color: #dc3545;
}

.banner-dismiss {
  background: none;
  border: none;
  color: inherit;
  cursor: pointer;
  opacity: 0.7;
  font-size: 0.8rem;
}

.banner-dismiss:hover {
  opacity: 1;
}

/* Tab content */
.tab-content {
  min-height: 200px;
}

.tab-actions {
  display: flex;
  gap: 0.5rem;
  margin-bottom: 1rem;
}

.tab-loading {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 3rem 0;
  gap: 1rem;
  color: var(--text-secondary);
}

/* Empty state */
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 4rem 2rem;
  gap: 1rem;
  color: var(--text-secondary);
  text-align: center;
}

.empty-state svg {
  opacity: 0.4;
}

.empty-state-inline {
  padding: 1.5rem;
  color: var(--text-secondary);
  text-align: center;
  background: var(--bg-secondary);
  border-radius: 8px;
}

.empty-state-inline p {
  margin: 0;
}

/* Card grid */
.card-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 1rem;
}

.resource-card {
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 10px;
  padding: 1.25rem;
  transition: border-color 0.2s;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.resource-card:hover {
  border-color: var(--accent-purple);
}

.card-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 0.75rem;
}

.card-description {
  color: var(--text-secondary);
  font-size: 0.875rem;
  line-height: 1.4;
  margin: 0;
}

.card-meta {
  display: flex;
  gap: 0.5rem;
  font-size: 0.8rem;
  color: var(--text-secondary);
}

.meta-label {
  font-weight: 500;
}

.meta-value {
  font-family: monospace;
  font-size: 0.75rem;
  opacity: 0.8;
}

.card-actions {
  display: flex;
  gap: 0.5rem;
  margin-top: auto;
  padding-top: 0.5rem;
}

/* Default skill toggle */
.resource-card.is-default-skill {
  border-color: var(--accent-purple);
  background: color-mix(in srgb, var(--accent-purple) 5%, var(--card-bg));
}

.card-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
  margin-top: auto;
  padding-top: 0.5rem;
}

.btn-default-toggle {
  background: transparent;
  border: 1px solid var(--border-color);
  color: var(--text-secondary);
  font-size: 0.75rem;
  padding: 0.3rem 0.7rem;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.2s;
  white-space: nowrap;
  flex-shrink: 0;
}

.btn-default-toggle:hover {
  border-color: var(--accent-purple);
  color: var(--accent-purple);
}

.btn-default-toggle.active {
  background: color-mix(in srgb, var(--accent-purple) 15%, transparent);
  border-color: var(--accent-purple);
  color: var(--accent-purple);
}

.btn-default-toggle:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-spinner-tiny {
  animation: spin 0.8s linear infinite;
}

.badge-default {
  background: var(--accent-purple) !important;
  color: white !important;
  font-size: 0.65rem;
  font-weight: 700;
  letter-spacing: 0.5px;
}

/* Badges */
.badge-row {
  display: flex;
  gap: 0.35rem;
  flex-shrink: 0;
  flex-wrap: wrap;
}

.badge {
  display: inline-flex;
  align-items: center;
  padding: 0.15rem 0.5rem;
  border-radius: 10px;
  font-size: 0.7rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.03em;
  white-space: nowrap;
}

.badge-default {
  background: var(--bg-secondary);
  color: var(--text-secondary);
}

.badge-project {
  background: rgba(99, 102, 241, 0.15);
  color: #818cf8;
}

.badge-personal {
  background: rgba(16, 185, 129, 0.15);
  color: #34d399;
}

.badge-effort {
  background: rgba(245, 158, 11, 0.15);
  color: #fbbf24;
}

.badge-installed {
  background: rgba(40, 167, 69, 0.15);
  color: #28a745;
}

.badge-category {
  background: rgba(99, 102, 241, 0.1);
  color: #a5b4fc;
}

.badge-command {
  background: rgba(59, 130, 246, 0.15);
  color: #60a5fa;
}

.badge-prompt {
  background: rgba(168, 85, 247, 0.15);
  color: #c084fc;
}

.badge-http {
  background: rgba(245, 158, 11, 0.15);
  color: #fbbf24;
}

.badge-agent {
  background: rgba(236, 72, 153, 0.15);
  color: #f472b6;
}

.badge-global {
  background: rgba(75, 85, 99, 0.3);
  color: #9ca3af;
}

.badge-local {
  background: rgba(16, 185, 129, 0.15);
  color: #34d399;
}

/* Skills */
.skill-name {
  font-family: 'SF Mono', 'Fira Code', monospace;
  font-size: 0.95rem;
  font-weight: 600;
  color: var(--text-primary);
}

/* Hooks */
.hooks-list {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.hook-event-group {
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 10px;
  overflow: hidden;
}

.hook-event-header {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  width: 100%;
  padding: 0.875rem 1.25rem;
  background: none;
  border: none;
  color: var(--text-primary);
  font-size: 0.95rem;
  font-weight: 600;
  cursor: pointer;
  text-align: left;
  transition: background 0.15s;
}

.hook-event-header:hover {
  background: var(--bg-secondary);
}

.chevron {
  transition: transform 0.2s;
  flex-shrink: 0;
}

.chevron.rotated {
  transform: rotate(90deg);
}

.event-name {
  flex: 1;
}

.event-count {
  background: var(--bg-secondary);
  color: var(--text-secondary);
  font-size: 0.75rem;
  font-weight: 600;
  padding: 0.1rem 0.5rem;
  border-radius: 10px;
  min-width: 1.5rem;
  text-align: center;
}

.hook-event-body {
  border-top: 1px solid var(--border-color);
  padding: 0.5rem 0;
}

.hook-item {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  padding: 0.75rem 1.25rem 0.75rem 2.75rem;
}

.hook-item + .hook-item {
  border-top: 1px solid var(--border-color);
}

.hook-badges {
  display: flex;
  gap: 0.35rem;
}

.hook-detail {
  font-size: 0.85rem;
}

.hook-command {
  font-family: 'SF Mono', 'Fira Code', monospace;
  font-size: 0.8rem;
  color: var(--text-secondary);
  background: var(--bg-secondary);
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
  word-break: break-all;
}

.hook-prompt {
  color: var(--text-secondary);
  font-style: italic;
}

/* MCP */
.mcp-section {
  margin-bottom: 2rem;
}

.section-title {
  font-size: 1.1rem;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0 0 1rem 0;
  padding-bottom: 0.5rem;
  border-bottom: 1px solid var(--border-color);
}

.mcp-name {
  font-weight: 600;
  font-size: 0.95rem;
  color: var(--text-primary);
}

.mcp-command {
  font-family: 'SF Mono', 'Fira Code', monospace;
  font-size: 0.8rem;
  color: var(--text-secondary);
  word-break: break-all;
}

.mcp-card .card-description {
  min-height: 1.5rem;
}

/* Add MCP form */
.add-mcp-form {
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 10px;
  padding: 1.5rem;
  margin-bottom: 1.5rem;
}

.add-mcp-form h3 {
  margin: 0 0 1rem 0;
  font-size: 1rem;
  color: var(--text-primary);
}

.form-group {
  margin-bottom: 1rem;
}

.form-group label {
  display: block;
  margin-bottom: 0.35rem;
  font-size: 0.85rem;
  font-weight: 500;
  color: var(--text-secondary);
}

.form-input {
  width: 100%;
  padding: 0.6rem 0.75rem;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 6px;
  color: var(--text-primary);
  font-size: 0.9rem;
  font-family: 'SF Mono', 'Fira Code', monospace;
  outline: none;
  transition: border-color 0.2s;
  box-sizing: border-box;
}

.form-input:focus {
  border-color: var(--accent-purple);
}

.form-input::placeholder {
  color: var(--text-secondary);
  opacity: 0.6;
}

.form-actions {
  display: flex;
  gap: 0.5rem;
  margin-top: 1rem;
}

/* Packs */
.pack-title-row {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.pack-icon {
  font-size: 1.25rem;
}

.pack-name {
  font-weight: 600;
  font-size: 0.95rem;
  color: var(--text-primary);
}

.pack-stats {
  display: flex;
  gap: 0.75rem;
  font-size: 0.8rem;
  color: var(--text-secondary);
}

.pack-stat {
  display: flex;
  align-items: center;
  gap: 0.25rem;
}

.pack-details-toggle {
  background: none;
  border: none;
  color: var(--accent-purple);
  font-size: 0.8rem;
  cursor: pointer;
  padding: 0;
  text-align: left;
}

.pack-details-toggle:hover {
  text-decoration: underline;
}

.pack-details {
  padding-top: 0.5rem;
  border-top: 1px solid var(--border-color);
}

.pack-detail-section h4 {
  margin: 0.5rem 0 0.35rem 0;
  font-size: 0.8rem;
  font-weight: 600;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.pack-detail-section ul {
  margin: 0;
  padding-left: 1.25rem;
  list-style: disc;
}

.pack-detail-section li {
  font-size: 0.8rem;
  color: var(--text-secondary);
  margin-bottom: 0.2rem;
}

.pack-detail-section code {
  font-family: 'SF Mono', 'Fira Code', monospace;
  font-size: 0.75rem;
}

/* Buttons */
.btn-action {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 1rem;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  color: var(--text-primary);
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-action:hover:not(:disabled) {
  border-color: var(--accent-purple);
  background: var(--card-bg);
}

.btn-action:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.btn-primary,
.btn-primary-small {
  background: var(--accent-purple);
  color: #fff;
  border: none;
  border-radius: 6px;
  font-weight: 500;
  cursor: pointer;
  transition: opacity 0.2s;
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
}

.btn-primary {
  padding: 0.6rem 1.25rem;
  font-size: 0.9rem;
}

.btn-primary-small {
  padding: 0.35rem 0.75rem;
  font-size: 0.8rem;
}

.btn-primary:hover:not(:disabled),
.btn-primary-small:hover:not(:disabled) {
  opacity: 0.85;
}

.btn-primary:disabled,
.btn-primary-small:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-secondary {
  padding: 0.6rem 1.25rem;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 6px;
  color: var(--text-primary);
  font-size: 0.9rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-secondary:hover {
  border-color: var(--accent-purple);
}

.btn-danger-small {
  padding: 0.35rem 0.75rem;
  background: rgba(220, 53, 69, 0.1);
  color: #dc3545;
  border: 1px solid rgba(220, 53, 69, 0.25);
  border-radius: 6px;
  font-size: 0.8rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-danger-small:hover:not(:disabled) {
  background: rgba(220, 53, 69, 0.2);
  border-color: rgba(220, 53, 69, 0.5);
}

.btn-danger-small:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* Spinner */
.btn-spinner-small {
  width: 14px;
  height: 14px;
  border: 2px solid var(--border-color);
  border-top-color: var(--accent-purple);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

/* Transitions */
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.25s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

.collapse-enter-active,
.collapse-leave-active {
  transition: all 0.25s ease;
  overflow: hidden;
}

.collapse-enter-from,
.collapse-leave-to {
  opacity: 0;
  max-height: 0;
  padding-top: 0;
  padding-bottom: 0;
}

/* Responsive */
@media (max-width: 768px) {
  .library-page {
    padding: 1rem;
  }

  .card-grid {
    grid-template-columns: 1fr;
  }

  .tab-bar {
    overflow-x: auto;
    -webkit-overflow-scrolling: touch;
  }

  .tab-btn {
    padding: 0.6rem 0.75rem;
    font-size: 0.85rem;
    white-space: nowrap;
  }

  .header-info h1 {
    font-size: 1.35rem;
  }

  .hook-item {
    padding-left: 1.25rem;
  }

  .form-actions {
    flex-direction: column;
  }
}

/* Templates section */
.templates-section {
  margin-bottom: 2rem;
  padding-bottom: 1.5rem;
  border-bottom: 1px solid var(--border-color);
}

.templates-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 1rem;
}

.templates-count {
  font-size: 0.8rem;
  color: var(--text-secondary);
}

.installed-skills-section {
  margin-top: 0.5rem;
}

.btn-templates {
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.1);
  border-color: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.3);
  color: rgb(167, 130, 255);
}

.btn-templates:hover:not(:disabled) {
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.2);
  border-color: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.5);
}

/* Category tabs */
.category-tabs {
  display: flex;
  gap: 0.5rem;
  flex-wrap: wrap;
  margin-bottom: 1rem;
}

.category-tab {
  padding: 0.35rem 0.75rem;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 20px;
  color: var(--text-secondary);
  font-size: 0.8rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
}

.category-tab:hover {
  border-color: var(--accent-purple);
  color: var(--text-primary);
}

.category-tab.active {
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.15);
  border-color: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.4);
  color: rgb(167, 130, 255);
}

.cat-count {
  font-size: 0.7rem;
  opacity: 0.7;
}

/* Template card specific */
.template-card {
  cursor: pointer;
  position: relative;
}

.template-card:hover {
  border-color: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.5);
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.05);
}

.template-install-hint {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  font-size: 0.75rem;
  color: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.6);
  margin-top: auto;
  padding-top: 0.5rem;
  transition: color 0.2s;
}

.template-card:hover .template-install-hint {
  color: rgb(139, 92, 246);
}

.section-title {
  font-size: 1rem;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0 0 1rem 0;
}

/* ========== Refero Design Styles ========== */

.scheme-filters {
  display: flex;
  gap: 0.5rem;
}

.refero-grid {
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
}

.refero-card {
  cursor: pointer;
  padding: 0;
  overflow: hidden;
  gap: 0;
}

.refero-card:hover {
  border-color: var(--accent-cyan, var(--accent-purple));
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}

.refero-thumbnail {
  width: 100%;
  aspect-ratio: 16 / 10;
  overflow: hidden;
  background: var(--bg-secondary);
  border-bottom: 1px solid var(--border-color);
}

.refero-thumbnail img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  transition: transform 0.3s;
}

.refero-card:hover .refero-thumbnail img {
  transform: scale(1.03);
}

.refero-thumbnail-placeholder {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 2.5rem;
  font-weight: 700;
  color: var(--text-secondary);
  background: linear-gradient(135deg, var(--bg-secondary) 0%, var(--bg-tertiary, var(--bg-secondary)) 100%);
}

.refero-info {
  padding: 1rem;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.refero-site-name {
  font-weight: 600;
  font-size: 0.95rem;
  color: var(--text-primary);
}

.refero-north-star {
  font-style: italic;
  font-size: 0.8rem;
  line-height: 1.35;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.badge-dark-scheme {
  background: rgba(30, 30, 40, 0.8);
  color: #c4c4cc;
  border: 1px solid rgba(100, 100, 120, 0.3);
}

.badge-light-scheme {
  background: rgba(255, 255, 240, 0.8);
  color: #555;
  border: 1px solid rgba(200, 200, 180, 0.4);
}

/* Color swatches row */
.refero-swatches {
  display: flex;
  align-items: center;
  gap: 4px;
}

.refero-swatch {
  width: 20px;
  height: 20px;
  border-radius: 4px;
  border: 1px solid rgba(128, 128, 128, 0.2);
  flex-shrink: 0;
}

.refero-swatch-more {
  font-size: 0.7rem;
  color: var(--text-secondary);
  margin-left: 2px;
}

/* Font tags */
.refero-fonts {
  display: flex;
  gap: 0.35rem;
  flex-wrap: wrap;
}

.refero-font-tag {
  font-size: 0.7rem;
  padding: 0.1rem 0.45rem;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 4px;
  color: var(--text-secondary);
  font-family: monospace;
}

/* Load more */
.refero-load-more {
  display: flex;
  justify-content: center;
  padding: 2rem 0 1rem;
}

/* ===== Refero Preview Modal ===== */

.refero-modal-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.6);
  backdrop-filter: blur(4px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  padding: 2rem;
}

.refero-modal {
  background: var(--bg-primary);
  border: 1px solid var(--border-color);
  border-radius: 14px;
  max-width: 640px;
  width: 100%;
  max-height: 85vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
}

.refero-modal-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  padding: 1.25rem 1.5rem;
  border-bottom: 1px solid var(--border-color);
  gap: 1rem;
}

.refero-modal-title-row {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.refero-modal-icon {
  width: 32px;
  height: 32px;
  border-radius: 6px;
  object-fit: contain;
}

.refero-modal-header h2 {
  margin: 0;
  font-size: 1.15rem;
  color: var(--text-primary);
}

.refero-modal-url {
  font-size: 0.8rem;
  color: var(--text-secondary);
  text-decoration: none;
}

.refero-modal-url:hover {
  color: var(--accent-purple);
  text-decoration: underline;
}

.refero-modal-close {
  background: none;
  border: none;
  font-size: 1.5rem;
  color: var(--text-secondary);
  cursor: pointer;
  padding: 0 0.25rem;
  line-height: 1;
  flex-shrink: 0;
}

.refero-modal-close:hover {
  color: var(--text-primary);
}

.refero-modal-body {
  padding: 1.5rem;
  overflow-y: auto;
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.refero-modal-screenshot {
  border-radius: 8px;
  overflow: hidden;
  border: 1px solid var(--border-color);
}

.refero-modal-screenshot img {
  width: 100%;
  display: block;
}

.refero-modal-north-star {
  font-style: italic;
  font-size: 1rem;
  color: var(--text-secondary);
  text-align: center;
  padding: 0.5rem 1rem;
  margin: 0;
}

.refero-modal-meta {
  display: flex;
  gap: 1rem;
  flex-wrap: wrap;
}

.refero-modal-meta-item {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.refero-modal-section h3 {
  font-size: 0.85rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--text-secondary);
  margin: 0 0 0.75rem 0;
}

.refero-modal-colors {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
}

.refero-modal-color {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.refero-modal-color-swatch {
  width: 28px;
  height: 28px;
  border-radius: 6px;
  border: 1px solid rgba(128, 128, 128, 0.2);
  flex-shrink: 0;
}

.refero-modal-color-info {
  display: flex;
  flex-direction: column;
}

.refero-modal-color-info code {
  font-size: 0.75rem;
  font-family: 'SF Mono', 'Fira Code', monospace;
  color: var(--text-primary);
}

.refero-modal-color-name {
  font-size: 0.7rem;
  color: var(--text-secondary);
}

.refero-modal-fonts {
  display: flex;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.refero-font-tag-large {
  font-size: 0.85rem;
  padding: 0.3rem 0.75rem;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 6px;
  color: var(--text-primary);
  font-weight: 500;
}

.refero-modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  padding: 1rem 1.5rem;
  border-top: 1px solid var(--border-color);
}

@media (max-width: 768px) {
  .refero-grid {
    grid-template-columns: 1fr;
  }

  .refero-modal-backdrop {
    padding: 1rem;
  }

  .refero-modal {
    max-height: 90vh;
  }

  .refero-modal-colors {
    gap: 0.5rem;
  }
}
</style>

