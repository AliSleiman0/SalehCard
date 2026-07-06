import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import type { Product } from '@/types'

// Stub the design-system barrel so the test renders ProductThumb in isolation
// (no i18n / heavy component tree). ProductThumb only pulls Art + artForCategory.
vi.mock('@/components', () => ({
  Art: ({ art }: { art: string }) => <div data-testid="art-placeholder" data-art={art} />,
  artForCategory: (category: string) => `art:${category}`,
}))

import { ProductThumb } from './ProductListPage'

function makeProduct(overrides: Partial<Product>): Product {
  return { category: 'gaming', images: [], ...overrides } as unknown as Product
}

// The <img> is decorative (alt=""), so it has no "img" a11y role — query the DOM.
const img = (c: HTMLElement) => c.querySelector('img')

describe('ProductThumb', () => {
  it('renders the thumbnail when present', () => {
    const { container } = render(
      <ProductThumb p={makeProduct({ thumbnail: 'http://cdn/t_thumb.jpg', images: ['http://cdn/t.jpg'] })} />,
    )
    expect(img(container)?.getAttribute('src')).toBe('http://cdn/t_thumb.jpg')
    expect(screen.queryByTestId('art-placeholder')).toBeNull()
  })

  it('falls back to images[0] when there is no thumbnail', () => {
    const { container } = render(<ProductThumb p={makeProduct({ images: ['http://cdn/display.jpg'] })} />)
    expect(img(container)?.getAttribute('src')).toBe('http://cdn/display.jpg')
  })

  it('renders the category placeholder when there is no image at all', () => {
    const { container } = render(<ProductThumb p={makeProduct({ category: 'topup', images: [] })} />)
    expect(img(container)).toBeNull()
    expect(screen.getByTestId('art-placeholder')).toHaveAttribute('data-art', 'art:topup')
  })

  // Legacy-import products come back with images: null (regression: this
  // crashed the whole /products page with "Cannot read properties of null").
  it('renders the placeholder when images is null', () => {
    const { container } = render(<ProductThumb p={makeProduct({ category: 'dd-live', images: null })} />)
    expect(img(container)).toBeNull()
    expect(screen.getByTestId('art-placeholder')).toHaveAttribute('data-art', 'art:dd-live')
  })

  it('swaps to the placeholder when the image URL fails to load', () => {
    const { container } = render(<ProductThumb p={makeProduct({ thumbnail: 'http://cdn/broken.jpg', images: [] })} />)
    const el = img(container)
    expect(el).not.toBeNull()
    fireEvent.error(el!)
    expect(img(container)).toBeNull()
    expect(screen.getByTestId('art-placeholder')).toBeInTheDocument()
  })
})
