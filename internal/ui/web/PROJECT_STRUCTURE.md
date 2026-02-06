# Mir Cockpit - Project Structure Guide

This document describes the best practices and project structure for the Mir Cockpit SvelteKit application.

## Table of Contents

- [Overview](#overview)
- [Directory Structure](#directory-structure)
- [Type System](#type-system)
- [State Management](#state-management)
- [Routing](#routing)
- [Services](#services)
- [Best Practices](#best-practices)

## Overview

**Tech Stack:**
- **Framework:** SvelteKit with Svelte 5
- **Build Tool:** Vite 7.3.1
- **UI Framework:** shadcn-svelte + bits-ui
- **Styling:** Tailwind CSS 4
- **Icons:** Lucide Svelte
- **TypeScript:** Strict mode enabled
- **Deployment:** Static adapter (SSR disabled, SPA mode)

## Directory Structure

```
internal/ui/web/
├── src/
│   ├── lib/
│   │   ├── components/         # Reusable UI components
│   │   │   ├── ui/            # shadcn-svelte components
│   │   │   ├── app-sidebar/   # Custom sidebar components
│   │   │   └── icons/         # Custom icon components
│   │   ├── types/             # TypeScript type definitions
│   │   │   ├── user.ts        # User-related types
│   │   │   ├── navigation.ts  # Navigation/sidebar types
│   │   │   ├── device.ts      # Device-related types
│   │   │   ├── api.ts         # API response types
│   │   │   └── index.ts       # Barrel export
│   │   ├── stores/            # Svelte 5 state management
│   │   │   ├── user.svelte.ts      # User authentication state
│   │   │   ├── theme.svelte.ts     # Theme preference state
│   │   │   └── sidebar.svelte.ts   # Sidebar state
│   │   ├── services/          # API client services
│   │   │   ├── api.ts         # Base HTTP client
│   │   │   └── devices.ts     # Device API service
│   │   ├── constants/         # Application constants
│   │   │   ├── routes.ts      # Route path constants
│   │   │   ├── config.ts      # App configuration
│   │   │   └── index.ts       # Barrel export
│   │   ├── utils/             # Utility functions
│   │   │   └── utils.ts       # cn() and other helpers
│   │   ├── data/              # Mock/static data
│   │   ├── hooks/             # Custom hooks
│   │   └── assets/            # Images, logos, etc.
│   ├── routes/
│   │   ├── (app)/             # Authenticated app routes
│   │   │   ├── +layout.svelte # App layout with sidebar
│   │   │   ├── +layout.ts     # Layout configuration
│   │   │   ├── dashboard/     # Dashboard page
│   │   │   ├── devices/       # Device management pages
│   │   │   ├── schemas/       # Schema pages
│   │   │   └── events/        # Event pages
│   │   ├── +layout.svelte     # Root layout
│   │   ├── +layout.ts         # Root configuration
│   │   ├── +page.svelte       # Home page (redirects to dashboard)
│   │   └── +error.svelte      # Error boundary
│   └── app.d.ts               # Global type declarations
├── static/                    # Static assets
└── [config files]
```

## Type System

All types are centralized in `src/lib/types/` for easy maintenance and reuse.

### Usage

```typescript
// Import types from the barrel export
import type { User, Device, NavItem, ApiResponse } from '$lib/types';
```

### Key Type Files

- **user.ts** - User authentication and profile types
- **navigation.ts** - Sidebar navigation types
- **device.ts** - IoT device types
- **api.ts** - API response and error types

## State Management

We use **Svelte 5 runes** for reactive state management.

### Store Pattern

```typescript
// lib/stores/user.svelte.ts
class UserStore {
  user = $state<User | null>(null);
  isLoading = $state(false);

  get isAuthenticated(): boolean {
    return this.user !== null;
  }

  async login(email: string, password: string) {
    this.isLoading = true;
    // ... login logic
  }
}

export const userStore = new UserStore();
```

### Usage in Components

```svelte
<script lang="ts">
  import { userStore } from '$lib/stores/user.svelte';
</script>

{#if userStore.user}
  <p>Welcome, {userStore.user.name}</p>
{/if}
```

### Available Stores

- **userStore** - User authentication state
- **themeStore** - Theme preference (light/dark/system)
- **sidebarStore** - Sidebar open/closed state

## Routing

We use **route groups** to organize pages with different layouts.

### Route Groups

- **(app)/** - Authenticated pages with sidebar
  - `/dashboard` - Main dashboard
  - `/devices` - Device management
  - `/schemas` - Schema editor
  - `/events` - Event viewer

Route groups (parentheses) don't affect URLs - `/dashboard` is at `(app)/dashboard/+page.svelte`.

### Creating New Routes

1. Create directory in `(app)/`
2. Add `+page.svelte` for the page
3. Optionally add `+page.ts` for data loading
4. Update `ROUTES` constant in `lib/constants/routes.ts`
5. Update sidebar navigation in `lib/data/sidebar-data.ts`

### Example: Adding a Settings Page

```typescript
// lib/constants/routes.ts
export const ROUTES = {
  // ... existing routes
  SETTINGS: '/settings'
} as const;

// lib/data/sidebar-data.ts
navMain: [
  // ... existing items
  {
    title: 'Settings',
    url: '/settings',
    icon: SettingsIcon
  }
]
```

```svelte
<!-- routes/(app)/settings/+page.svelte -->
<script lang="ts">
  import * as Card from '$lib/components/ui/card';
</script>

<div class="space-y-4">
  <h1 class="text-3xl font-bold">Settings</h1>
  <Card.Root>
    <Card.Content>
      Settings content here
    </Card.Content>
  </Card.Root>
</div>
```

## Services

API services provide a clean interface for backend communication.

### Base API Client

Located at `lib/services/api.ts`, provides:
- GET, POST, PUT, PATCH, DELETE methods
- Automatic timeout handling
- Consistent error handling
- Type-safe responses

### Usage

```typescript
import { deviceService } from '$lib/services/devices';

// In a +page.ts file
export async function load() {
  const response = await deviceService.getAll();
  return { devices: response.data };
}
```

### Creating New Services

```typescript
// lib/services/schemas.ts
import { api } from './api';
import type { Schema, ApiResponse } from '$lib/types';

export const schemaService = {
  async getAll(): Promise<ApiResponse<Schema[]>> {
    return api.get<Schema[]>('/schemas');
  },

  async create(schema: SchemaInput): Promise<ApiResponse<Schema>> {
    return api.post<Schema>('/schemas', schema);
  }
};
```

## Best Practices

### Component Organization

```svelte
<script lang="ts">
  // 1. Imports
  import type { User } from '$lib/types';
  import { Button } from '$lib/components/ui/button';

  // 2. Props
  let { user }: { user: User } = $props();

  // 3. Local state
  let count = $state(0);

  // 4. Derived state
  let doubled = $derived(count * 2);

  // 5. Effects
  $effect(() => {
    console.log(`Count: ${count}`);
  });

  // 6. Functions
  function handleClick() {
    count++;
  }
</script>

<!-- Template -->
<button onclick={handleClick}>
  {user.name}: {doubled}
</button>
```

### File Naming

- **Components:** PascalCase (e.g., `AppSidebar.svelte`)
- **Routes:** kebab-case (e.g., `device-detail/`)
- **Utilities:** camelCase (e.g., `formatDate.ts`)
- **Types:** PascalCase (e.g., `User`, `Device`)
- **Constants:** UPPER_SNAKE_CASE (e.g., `API_BASE_URL`)

### Import Paths

Always use the `$lib` alias for library imports:

```typescript
// ✅ Good
import { Button } from '$lib/components/ui/button';
import type { User } from '$lib/types';
import { ROUTES } from '$lib/constants';

// ❌ Bad
import { Button } from '../../lib/components/ui/button';
```

### TypeScript

- Use `type` for type definitions (not `interface`)
- Always type component props
- Use type imports: `import type { ... }`
- Leverage type inference where possible

### Styling

- Use Tailwind utility classes
- Use `tailwind-variants` for component variants
- Keep custom CSS minimal
- Use CSS variables for theming (already configured)

### Data Fetching

```typescript
// routes/(app)/devices/+page.ts
import type { PageLoad } from './$types';
import { deviceService } from '$lib/services/devices';

export const load: PageLoad = async () => {
  const response = await deviceService.getAll();
  return {
    devices: response.data.items,
    title: 'Devices'
  };
};
```

### Error Handling

The app has a global error boundary at `routes/+error.svelte` that catches:
- 404 Not Found errors
- 500 Server errors
- Unexpected errors

To throw expected errors:

```typescript
import { error } from '@sveltejs/kit';

export const load = async ({ params }) => {
  const device = await deviceService.getById(params.id);

  if (!device) {
    throw error(404, 'Device not found');
  }

  return { device };
};
```

## Next Steps

### Recommended Additions

1. **Authentication**
   - Implement actual login/logout flow
   - Add protected route guards
   - Use `userStore` for auth state

2. **Testing**
   - Add Vitest tests for services
   - Add Playwright E2E tests for critical flows

3. **Real API Integration**
   - Replace mock data with actual API calls
   - Implement proper error handling
   - Add loading states

4. **Performance**
   - Add code splitting for routes
   - Implement virtual scrolling for long lists
   - Optimize images and assets

## Resources

- [Svelte 5 Documentation](https://svelte.dev)
- [SvelteKit Documentation](https://kit.svelte.dev)
- [shadcn-svelte](https://github.com/shadcn-svelte/ui)
- [Tailwind CSS](https://tailwindcss.com)
