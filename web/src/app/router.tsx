import { Suspense, lazy } from 'react'
import { Routes, Route } from 'react-router-dom'
import Layout from './Layout'
import { RequireAuth, RequireReseller, RedirectIfAuthed } from './RequireAuth'

const HomePage = lazy(() => import('@/features/catalog/pages/HomePage'))
const CategoryPage = lazy(() => import('@/features/catalog/pages/CategoryPage'))
const ProductDetailPage = lazy(() => import('@/features/catalog/pages/ProductDetailPage'))
const CartPage = lazy(() => import('@/features/cart/pages/CartPage'))
const CheckoutPage = lazy(() => import('@/features/checkout/pages/CheckoutPage'))
const OrderSuccessPage = lazy(() => import('@/features/orders/pages/OrderSuccessPage'))
const LoginPage = lazy(() => import('@/features/auth/pages/LoginPage'))
const RegisterPage = lazy(() => import('@/features/auth/pages/RegisterPage'))
const DashboardPage = lazy(() => import('@/features/auth/pages/DashboardPage'))
const WalletPage = lazy(() => import('@/features/wallet/pages/WalletPage'))
const OrdersPage = lazy(() => import('@/features/orders/pages/OrdersPage'))
const OrderDetailPage = lazy(() => import('@/features/orders/pages/OrderDetailPage'))
const SavedIDsPage = lazy(() => import('@/features/auth/pages/SavedIDsPage'))
const ResellerDashboardPage = lazy(() => import('@/features/reseller/pages/ResellerDashboardPage'))
const KycPage = lazy(() => import('@/features/kyc/pages/KycPage'))
const KycFormPage = lazy(() => import('@/features/kyc/pages/KycFormPage'))
const SearchPage = lazy(() => import('@/features/catalog/pages/SearchPage'))
const OffersPage = lazy(() => import('@/features/offers/pages/OffersPage'))

const fallback = (
  <div className="flex items-center justify-center h-screen">Loading...</div>
)

export default function AppRouter() {
  return (
    <Suspense fallback={fallback}>
      <Routes>
        <Route element={<Layout />}>
          <Route path="/" element={<HomePage />} />
          <Route path="/category/:slug" element={<CategoryPage />} />
          <Route path="/product/:id" element={<ProductDetailPage />} />
          <Route path="/search" element={<SearchPage />} />
          <Route
            path="/offers"
            element={
              <RequireAuth>
                <OffersPage />
              </RequireAuth>
            }
          />
          <Route path="/cart" element={<CartPage />} />
          <Route
            path="/checkout"
            element={
              <RequireAuth>
                <CheckoutPage />
              </RequireAuth>
            }
          />
          <Route
            path="/order-success/:orderId"
            element={
              <RequireAuth>
                <OrderSuccessPage />
              </RequireAuth>
            }
          />
          <Route
            path="/login"
            element={
              <RedirectIfAuthed>
                <LoginPage />
              </RedirectIfAuthed>
            }
          />
          <Route
            path="/register"
            element={
              <RedirectIfAuthed>
                <RegisterPage />
              </RedirectIfAuthed>
            }
          />
          <Route
            path="/dashboard"
            element={
              <RequireAuth>
                <DashboardPage />
              </RequireAuth>
            }
          />
          <Route
            path="/wallet"
            element={
              <RequireAuth>
                <WalletPage />
              </RequireAuth>
            }
          />
          <Route
            path="/orders"
            element={
              <RequireAuth>
                <OrdersPage />
              </RequireAuth>
            }
          />
          <Route
            path="/orders/:id"
            element={
              <RequireAuth>
                <OrderDetailPage />
              </RequireAuth>
            }
          />
          <Route
            path="/saved-ids"
            element={
              <RequireAuth>
                <SavedIDsPage />
              </RequireAuth>
            }
          />
          <Route
            path="/kyc"
            element={
              <RequireAuth>
                <KycPage />
              </RequireAuth>
            }
          />
          <Route
            path="/kyc/submit"
            element={
              <RequireAuth>
                <KycFormPage />
              </RequireAuth>
            }
          />
          <Route
            path="/reseller"
            element={
              <RequireReseller>
                <ResellerDashboardPage />
              </RequireReseller>
            }
          />
        </Route>
      </Routes>
    </Suspense>
  )
}
