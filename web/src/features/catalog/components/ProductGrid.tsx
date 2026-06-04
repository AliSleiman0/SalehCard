import { ProductCard } from './ProductCard'
import type { ViewProduct } from '../lib/adaptProduct'

export function ProductGrid({ products }: { products: ViewProduct[] }) {
  return (
    <div className="prodgrid">
      {products.map((p) => (
        <ProductCard key={p.id} p={p} />
      ))}
    </div>
  )
}
