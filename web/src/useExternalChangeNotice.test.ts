import { renderHook, act } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useExternalChangeNotice } from './useExternalChangeNotice'

class FakeWebSocket {
  static instances: FakeWebSocket[] = []
  onmessage: ((event: { data: string }) => void) | null = null
  closed = false
  url: string

  constructor(url: string) {
    this.url = url
    FakeWebSocket.instances.push(this)
  }

  close() {
    this.closed = true
  }

  emit(data: unknown) {
    this.onmessage?.({ data: JSON.stringify(data) })
  }
}

describe('useExternalChangeNotice', () => {
  beforeEach(() => {
    FakeWebSocket.instances = []
    vi.stubGlobal('WebSocket', FakeWebSocket)
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('flips to true when a matching path event arrives', () => {
    const { result } = renderHook(() => useExternalChangeNotice('Hub.md'))
    expect(result.current[0]).toBe(false)

    act(() => {
      FakeWebSocket.instances[0].emit({ path: 'Hub.md' })
    })

    expect(result.current[0]).toBe(true)
  })

  it('ignores events for a different path', () => {
    const { result } = renderHook(() => useExternalChangeNotice('Hub.md'))

    act(() => {
      FakeWebSocket.instances[0].emit({ path: 'Other.md' })
    })

    expect(result.current[0]).toBe(false)
  })

  it('reset() flips it back to false', () => {
    const { result } = renderHook(() => useExternalChangeNotice('Hub.md'))

    act(() => {
      FakeWebSocket.instances[0].emit({ path: 'Hub.md' })
    })
    expect(result.current[0]).toBe(true)

    act(() => {
      result.current[1]()
    })
    expect(result.current[0]).toBe(false)
  })

  it('closes the socket on unmount', () => {
    const { unmount } = renderHook(() => useExternalChangeNotice('Hub.md'))
    const ws = FakeWebSocket.instances[0]

    unmount()

    expect(ws.closed).toBe(true)
  })

  it('opens a new connection when path changes', () => {
    const { rerender } = renderHook(({ path }) => useExternalChangeNotice(path), {
      initialProps: { path: 'Hub.md' },
    })

    rerender({ path: 'Leaf.md' })

    expect(FakeWebSocket.instances).toHaveLength(2)
  })
})
