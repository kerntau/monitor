import { t } from "@lingui/core/macro"
import { useStore } from "@nanostores/react"
import type { AuthMethodsList } from "pocketbase"
import { useEffect, useMemo, useState } from "react"
import { UserAuthForm } from "@/components/login/auth-form"
import { pb } from "@/lib/api"
import { $router } from "../router"
import { LoginLayout } from "./layout"
import ForgotPassword from "./forgot-pass-form"
import { OtpRequestForm } from "./otp-forms"

export default function () {
	const page = useStore($router)
	const [isFirstRun, setFirstRun] = useState(false)
	const [authMethods, setAuthMethods] = useState<AuthMethodsList>()

	useEffect(() => {
		document.title = t`Login` + " / chenv"

		pb.send("/api/beszel/first-run", {}).then(({ firstRun }) => {
			setFirstRun(firstRun)
		})
	}, [])

	useEffect(() => {
		pb.collection("users")
			.listAuthMethods()
			.then((methods) => {
				setAuthMethods(methods)
			})
	}, [])

	const subtitle = useMemo(() => {
		if (isFirstRun) {
			return t`Please create an admin account`
		} else if (page?.route === "forgot_password") {
			return t`Enter email address to reset password`
		} else if (page?.route === "request_otp") {
			return t`Request a one-time password`
		} else {
			return t`Please sign in to your account`
		}
	}, [isFirstRun, page])

	if (!authMethods) {
		return null
	}

	return (
        <LoginLayout subtitle={subtitle}>
            {page?.route === "forgot_password" ? <ForgotPassword /> : page?.route === "request_otp" ? <OtpRequestForm /> : <UserAuthForm isFirstRun={isFirstRun} authMethods={authMethods} />}
        </LoginLayout>
    )
}
