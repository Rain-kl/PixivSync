"use client"

import {motion} from "motion/react"
import {usePathname, useRouter} from "next/navigation"
import {useEffect, useState} from "react"
import {AppSidebar} from "@/components/layout/sidebar"
import {SiteHeader} from "@/components/layout/header"
import {SidebarInset, SidebarProvider} from "@/components/ui/sidebar"
import {LoadingPage} from "@/components/layout/loading"
import {useUser} from "@/contexts/user-context"


export default function MainLayout({
  children,
}: {
  children: React.ReactNode
}) {
  const router = useRouter()
  const pathname = usePathname()
  const {user, loading} = useUser()
  const [isFullWidth, setIsFullWidth] = useState(false)

  useEffect(() => {
    if (loading || user) return

    const queryString = window.location.search
    const callbackUrl = queryString ? `${pathname}${queryString}` : pathname
    const loginUrl = new URL("/login", window.location.origin)

    loginUrl.searchParams.set("callbackUrl", callbackUrl)
    sessionStorage.setItem("redirect_after_login", callbackUrl)
    router.replace(loginUrl.toString())
  }, [loading, pathname, router, user])

  if (loading || !user) {
    return <LoadingPage text="登录状态" badgeText="Auth" />
  }

  return (
    <SidebarProvider
      className="h-screen"
      style={
        {
          "--header-height": "60px",
        } as React.CSSProperties
      }
    >
      <AppSidebar />
      <SidebarInset className="flex flex-col min-w-0 h-screen">
        <SiteHeader isFullWidth={isFullWidth} onToggleFullWidth={setIsFullWidth} />
        <div className="flex flex-1 flex-col bg-background overflow-y-auto overflow-x-hidden min-w-0 hide-scrollbar">
          <div className={`w-full mx-auto px-4 sm:px-6 md:px-8 lg:px-12 min-w-0 transition-all duration-300 ease-in-out ${!isFullWidth ? "max-w-[1320px]" : "max-w-full"}`}>
            <motion.div
              key={pathname}
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              transition={{
                duration: 0.5,
                ease: "easeOut",
              }}
              className="w-full"
            >
              {children}
            </motion.div>
          </div>
        </div>
      </SidebarInset>
    </SidebarProvider>
  )
}
