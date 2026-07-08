import { useQuery } from '@tanstack/react-query'
import { listSoldByProduct } from '../api/sold'

/** Units-sold-per-product read model for the product list "Sold" column. Keyed
 *  independently of the (paginated) product list so it is fetched once and reused
 *  across pages. */
export function useSoldByProduct(opts?: { enabled?: boolean }) {
  return useQuery({
    queryKey: ['admin', 'products', 'sold'],
    queryFn: () => listSoldByProduct(),
    // Reads the orders domain; skipped when the admin lacks orders.view.
    enabled: opts?.enabled ?? true,
  })
}
