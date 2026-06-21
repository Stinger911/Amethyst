import { cleanup } from '@testing-library/react'
import '@testing-library/jest-dom/vitest'
import { afterEach } from 'vitest'

// RTL's auto-cleanup only self-registers when it detects a global
// `afterEach` (i.e. vitest's `test.globals: true`); we don't enable that,
// so it's wired up explicitly here instead.
afterEach(cleanup)
