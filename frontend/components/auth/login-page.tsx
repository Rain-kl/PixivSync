"use client"

import {useCallback, useEffect, useState} from "react"
import {AnimatePresence, motion} from "motion/react"
import {useRouter, useSearchParams} from "next/navigation"
import {toast} from "sonner"
import {Spinner} from "@/components/ui/spinner"
import {LoginForm} from "@/components/auth/login-form"
import {Check} from "lucide-react"

import {AuroraBackground} from "@/components/ui/aurora-background"
import services from "@/lib/services"
import {useAuth} from "@/components/providers/auth-provider"


/**
 * 登录页面组件
 * 显示登录表单和登录按钮
 *
 * @example
 * ```tsx
 * <LoginPage />
 * ```
 * @returns {React.ReactNode} 登录页面组件
 */
export function LoginPage() {
  const router = useRouter()
  const searchParams = useSearchParams()
  const { setUser } = useAuth()

  /* 处理OAuth回调 */
  const [isProcessingCallback, setIsProcessingCallback] = useState(() => {
    const state = searchParams.get('state')
    const code = searchParams.get('code')
    return !!(state && code)
  })
  const [isCheckingSession, setIsCheckingSession] = useState(() => !searchParams.get('state') || !searchParams.get('code'))

  const [loginSuccess, setLoginSuccess] = useState(false)

  const resolveRedirectTarget = useCallback(() => {
    const callbackUrl = searchParams.get('callbackUrl')
    const storedRedirect = sessionStorage.getItem('redirect_after_login')
    const target = callbackUrl || storedRedirect || '/home'

    if (storedRedirect) {
      sessionStorage.removeItem('redirect_after_login')
    }

    return target
  }, [searchParams])

  /* 登录页兜底：已登录用户直接跳转 */
  useEffect(() => {
    const state = searchParams.get('state')
    const code = searchParams.get('code')

    if (state && code) {
      setIsCheckingSession(false)
      return
    }

    let cancelled = false

    const checkExistingSession = async () => {
      setIsCheckingSession(true)

      try {
        const response = await fetch('/api/v1/oauth/user-info', {
          credentials: 'include',
          cache: 'no-store',
        })

        if (cancelled) return

        if (response.ok) {
          const payload = await response.json()
          if (payload?.data) {
            setUser(payload.data)
          }
          router.replace(resolveRedirectTarget())
          return
        }
      } catch (error) {
        if (!cancelled) {
          console.error('Session probe error:', error)
        }
      } finally {
        if (!cancelled) {
          setIsCheckingSession(false)
        }
      }
    }

    checkExistingSession()

    return () => {
      cancelled = true
    }
  }, [router, searchParams, resolveRedirectTarget, setUser])

  /* 回调逻辑 */
  useEffect(() => {
    const handleOAuthCallback = async () => {
      const state = searchParams.get('state')
      const code = searchParams.get('code')

      if (state && code) {
        setIsProcessingCallback(true)
        try {
          const result = await services.auth.handleCallback({ state, code })
          if (result.status === "need_bind") {
            toast.info("您的第三方账号未绑定本地账号，系统已关闭注册。请登录已有本地账号进行绑定。")
            setIsProcessingCallback(false)
            router.replace('/login')
            return
          }
          if (result.user) {
            setUser(result.user)
          }
          setLoginSuccess(true)
          toast.success(result.status === "bound" ? "绑定成功" : "登录成功")

          setTimeout(() => {
            router.replace(resolveRedirectTarget())
          }, 1500)
        } catch (error) {
          console.error('OAuth callback error:', error)
          toast.error(error instanceof Error ? error.message : "登录失败，请重试")
          setIsProcessingCallback(false)
          router.replace('/login')
        }
      }
    }
    handleOAuthCallback()
  }, [searchParams, router, resolveRedirectTarget, setUser])

  return (
    <AuroraBackground>
      <motion.div
        initial={{ opacity: 0, y: 40 }}
        whileInView={{ opacity: 1, y: 0 }}
        viewport={{ once: true }}
        transition={{
          delay: 0.3,
          duration: 0.8,
          ease: "easeInOut",
        }}
        className="relative z-10 w-full max-w-sm px-4"
      >
        {/*Title Do Not Remove*/}
        {/*<div className="text-center mb-8 space-y-2">*/}
        {/*  <h1 className="text-3xl font-bold tracking-tight text-foreground">*/}
        {/*    Refresh*/}
        {/*  </h1>*/}
        {/*</div>*/}

        <AnimatePresence mode="wait">
          {isProcessingCallback || isCheckingSession ? (
            <motion.div
              key={isProcessingCallback ? "processing" : "session-check"}
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
              className="w-full"
            >
              {isCheckingSession ? (
                <div className="flex flex-col items-center justify-center space-y-4 py-2">
                  <div className="relative">
                    <Spinner className="w-8 h-8 text-blue-600" />
                  </div>
                  <div className="text-center space-y-2">
                    <h3 className="font-semibold tracking-tight text-foreground">正在检查登录状态</h3>
                    <p className="text-xs text-muted-foreground">请稍候，我们正在确认当前会话...</p>
                  </div>
                </div>
              ) : loginSuccess ? (
                <div className="flex flex-col items-center justify-center space-y-4 py-2">
                  <motion.div
                    initial={{ scale: 0.5, opacity: 0 }}
                    animate={{ scale: 1, opacity: 1 }}
                    transition={{ type: "spring", stiffness: 300, damping: 20 }}
                    className="w-8 h-8 rounded-full bg-green-500/10 flex items-center justify-center text-green-500 ring-1 ring-green-500/20"
                  >
                    <Check className="w-6 h-6" strokeWidth={3} />
                  </motion.div>
                  <div className="text-center space-y-2">
                    <h3 className="font-semibold tracking-tight text-foreground">登录成功</h3>
                    <p className="text-xs text-muted-foreground">正在跳转至控制台...</p>
                  </div>
                </div>
              ) : (
                <div className="flex flex-col items-center justify-center space-y-4 py-2">
                  <div className="relative">
                    <Spinner className="w-8 h-8 text-blue-600" />
                  </div>
                  <div className="text-center space-y-2">
                    <h3 className="font-semibold tracking-tight text-foreground">正在验证凭据</h3>
                    <p className="text-xs text-muted-foreground">请稍候，我们正在为您建立安全会话...</p>
                  </div>
                </div>
              )}
            </motion.div>
          ) : (
            <motion.div
              key="login-form-wrapper"
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              transition={{ duration: 0.4 }}
              className="w-full"
            >
              <LoginForm />
            </motion.div>
          )}
        </AnimatePresence>
      </motion.div>
    </AuroraBackground>
  )
}
