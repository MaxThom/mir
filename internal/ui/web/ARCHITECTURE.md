# Mir Cockpit Architecture

## System Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                         User Interface                           │
│                     (Svelte 5 Components)                        │
└───────────────────────────┬─────────────────────────────────────┘
                            │
                            ├─── Routes (pages)
                            ├─── Components (UI)
                            ├─── Stores (state)
                            └─── Services (API)
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Backend API Server                          │
│                     (To be implemented)                          │
└─────────────────────────────────────────────────────────────────┘
```

## Application Layers

### 1. Presentation Layer (Routes & Components)

```
routes/
├── (app)/                     # Authenticated routes with sidebar
│   ├── dashboard/            # Overview & stats
│   ├── devices/              # Device management
│   ├── schemas/              # Data schema editor
│   └── events/               # Event logs
└── +error.svelte             # Global error boundary
```

**Responsibility:** User interface, user interactions, display logic

### 2. State Management Layer (Stores)

```
lib/stores/
├── user.svelte.ts            # Authentication state
├── theme.svelte.ts           # UI theme (dark/light)
└── sidebar.svelte.ts         # Sidebar state
```

**Responsibility:** Application state, reactive data, shared state

**Pattern:**
```typescript
class Store {
  data = $state(initialValue);

  get computed() {
    return derivedValue;
  }

  methods() {
    // Mutate state
  }
}
```

### 3. Service Layer (API Clients)

```
lib/services/
├── api.ts                    # Base HTTP client
└── devices.ts                # Device API
```

**Responsibility:** Backend communication, data fetching, business logic

**Flow:**
```
Component → Service → API Server → Database
         ← Response ← Response  ←
```

### 4. Type System (TypeScript)

```
lib/types/
├── user.ts                   # User types
├── device.ts                 # Device types
├── navigation.ts             # Navigation types
├── api.ts                    # API types
└── index.ts                  # Exports
```

**Responsibility:** Type safety, contracts between layers

## Data Flow

### Reading Data (Query)

```
┌──────────┐     ┌─────────┐     ┌─────────┐     ┌─────────┐
│  Page    │────▶│ Service │────▶│   API   │────▶│ Backend │
│ +page.ts │     │         │     │ Client  │     │ Server  │
└──────────┘     └─────────┘     └─────────┘     └─────────┘
     │                                                   │
     │                                                   │
     ▼                                                   ▼
┌──────────┐                                      ┌─────────┐
│Component │◀────────────────────────────────────│Database │
│+page.svlt│         Data returned                └─────────┘
└──────────┘
```

### Writing Data (Mutation)

```
┌──────────┐     ┌─────────┐     ┌─────────┐     ┌─────────┐
│Component │────▶│ Service │────▶│   API   │────▶│ Backend │
│ (form)   │     │         │     │ Client  │     │ Server  │
└──────────┘     └─────────┘     └─────────┘     └─────────┘
     │                                                   │
     │                                                   │
     ▼                                                   ▼
┌──────────┐                                      ┌─────────┐
│  Store   │◀────────────────────────────────────│Database │
│ (state)  │      Update local state              └─────────┘
└──────────┘
```

## Component Architecture

### Page Component Pattern

```svelte
<!-- routes/(app)/devices/+page.svelte -->
<script lang="ts">
  import type { PageData } from './$types';
  import { deviceService } from '$lib/services/devices';
  import DeviceList from './DeviceList.svelte';

  // Data from +page.ts
  let { data }: { data: PageData } = $props();

  // Local UI state
  let searchQuery = $state('');

  // Derived state
  let filteredDevices = $derived(
    data.devices.filter(d => d.name.includes(searchQuery))
  );
</script>

<DeviceList devices={filteredDevices} />
```

### Reusable Component Pattern

```svelte
<!-- lib/components/DeviceCard.svelte -->
<script lang="ts">
  import type { Device } from '$lib/types';

  let {
    device,
    onSelect
  }: {
    device: Device;
    onSelect?: (device: Device) => void;
  } = $props();
</script>

<div onclick={() => onSelect?.(device)}>
  {device.name}
</div>
```

## State Management Patterns

### Local State (Component-specific)

```svelte
<script lang="ts">
  let isOpen = $state(false);
  let count = $state(0);
</script>
```

**Use for:** UI state, form inputs, temporary data

### Shared State (Cross-component)

```typescript
// lib/stores/cart.svelte.ts
class CartStore {
  items = $state<Item[]>([]);

  get total() {
    return this.items.reduce((sum, item) => sum + item.price, 0);
  }

  addItem(item: Item) {
    this.items.push(item);
  }
}

export const cartStore = new CartStore();
```

**Use for:** User auth, theme, global settings

### Server State (From API)

```typescript
// routes/(app)/devices/+page.ts
export const load = async ({ fetch }) => {
  const response = await deviceService.getAll();
  return { devices: response.data };
};
```

**Use for:** Backend data, persisted state

## Routing Architecture

### Route Groups

```
(app)/              # Group: authenticated routes
  ├── dashboard/    # URL: /dashboard
  ├── devices/      # URL: /devices
  └── schemas/      # URL: /schemas

(auth)/             # Group: public auth routes (future)
  ├── login/        # URL: /login
  └── register/     # URL: /register
```

**Benefits:**
- Different layouts per group
- Shared loading logic
- Easy to add auth guards

### Nested Routes

```
devices/
├── +page.svelte              # /devices (list)
├── [id]/
│   ├── +page.svelte         # /devices/123 (detail)
│   └── edit/
│       └── +page.svelte     # /devices/123/edit
└── create/
    └── +page.svelte         # /devices/create
```

## Error Handling Architecture

### Levels of Error Handling

1. **Component Level** - Try/catch in event handlers
2. **Service Level** - HTTP errors, timeouts
3. **Route Level** - `throw error(404)` in load functions
4. **Global Level** - `+error.svelte` boundary

### Error Flow

```
┌─────────────┐
│  Component  │───▶ try/catch ───▶ Show error message
└─────────────┘

┌─────────────┐
│   Service   │───▶ HTTP error ───▶ Throw ApiError
└─────────────┘

┌─────────────┐
│    Route    │───▶ throw error() ─▶ +error.svelte
└─────────────┘

┌─────────────┐
│   Global    │───▶ Any error ────▶ +error.svelte (fallback)
└─────────────┘
```

## File Organization Principles

### 1. Colocation
Keep related files together:
```
devices/
├── DeviceCard.svelte
├── DeviceList.svelte
├── DeviceForm.svelte
└── +page.svelte
```

### 2. Feature-Based Structure
Organize by feature, not file type:
```
✅ Good:
lib/
├── devices/
│   ├── DeviceCard.svelte
│   └── device-utils.ts
└── users/
    ├── UserCard.svelte
    └── user-utils.ts

❌ Bad:
lib/
├── components/
│   ├── DeviceCard.svelte
│   └── UserCard.svelte
└── utils/
    ├── device-utils.ts
    └── user-utils.ts
```

### 3. Shared Code Extraction
Common code goes in `lib/`:
```
lib/
├── components/ui/    # Shared UI components
├── utils/            # Shared utilities
├── types/            # Shared types
└── services/         # Shared API clients
```

## Build & Deployment

### Build Process

```
Source Code (TypeScript + Svelte)
       ↓
  Vite Build
       ↓
  SvelteKit Adapter (@sveltejs/adapter-static)
       ↓
  Static Files (build/)
       ↓
  Web Server (nginx/Apache/S3)
```

### Configuration

- **SSR:** Disabled (SPA mode)
- **Prerendering:** Enabled
- **Output:** Static HTML/JS/CSS
- **Target:** Modern browsers

## Performance Considerations

### Code Splitting

- Automatic per-route
- Dynamic imports for heavy components
- Lazy loading for icons/charts

### State Updates

- Svelte 5 runes are highly optimized
- Fine-grained reactivity
- No virtual DOM overhead

### Data Loading

- Parallel loading with `Promise.all()`
- Preloading on hover (`data-sveltekit-preload-data`)
- Caching in service layer (future)

## Security Architecture

### Client-Side Security

1. **Type Safety** - TypeScript prevents runtime errors
2. **Input Validation** - Zod schemas (to be added)
3. **XSS Prevention** - Svelte auto-escapes by default
4. **CSRF** - SvelteKit handles tokens (when SSR enabled)

### API Security (Backend)

- **Authentication** - JWT tokens (to be implemented)
- **Authorization** - Role-based access control
- **Rate Limiting** - API throttling
- **Input Validation** - Server-side validation

## Scalability Patterns

### Current Scale (Small Project)
- Single store per domain
- Direct API calls
- Client-side filtering

### Future Scale (Medium Project)
- Multiple stores with composition
- API client with caching
- Server-side pagination

### Enterprise Scale (Future)
- State management library (if needed)
- GraphQL or tRPC
- Micro-frontends

## Testing Strategy (Planned)

```
Unit Tests (Vitest)
├── Services (API mocking)
├── Stores (state logic)
└── Utils (pure functions)

Integration Tests (Testing Library)
├── Components (with props)
├── Forms (user interactions)
└── Navigation (route changes)

E2E Tests (Playwright)
├── Critical flows (login, create device)
├── Cross-browser testing
└── Accessibility testing
```

## Development Workflow

```
1. Define types (lib/types/)
   ↓
2. Create service (lib/services/)
   ↓
3. Build components (lib/components/)
   ↓
4. Wire up in routes (routes/)
   ↓
5. Add to navigation (sidebar-data.ts)
```

## Key Design Decisions

| Decision | Rationale |
|----------|-----------|
| Svelte 5 runes | Modern, type-safe state management |
| Route groups | Clean layout separation |
| Service layer | Separation of concerns |
| Centralized types | DRY, type safety |
| Static adapter | Fast, deployable anywhere |
| No SSR | Simplicity for SPA use case |
| TypeScript strict | Catch errors early |

## Future Architecture Goals

1. **Real-time Updates** - WebSocket integration
2. **Offline Support** - Service workers, local storage
3. **Internationalization** - i18n library
4. **Advanced Caching** - Query cache (TanStack Query)
5. **Monitoring** - Error tracking (Sentry)
6. **Analytics** - User behavior tracking
