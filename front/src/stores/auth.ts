import type { ApiError } from "@composables/useApi";
import type { AuthProfile } from "@composables/useAuth";
import { useAuth } from "@composables/useAuth";
import { defineStore } from "pinia";

type AuthStatus = "unknown" | "authenticated" | "guest";

interface AuthState {
	status: AuthStatus;
	profile: AuthProfile | null;
	isLoading: boolean;
}

export const useAuthStore = defineStore("auth", {
	state: (): AuthState => ({
		status: "unknown",
		profile: null,
		isLoading: false,
	}),
	getters: {
		isAuthenticated: (state) => state.status === "authenticated",
	},
	actions: {
		async restoreSession(): Promise<void> {
			if (this.status !== "unknown") {
				return;
			}

			this.isLoading = true;
			const { me } = useAuth();

			try {
				const profile = await me();
				this.profile = profile;
				this.status = "authenticated";
			} catch (error) {
				const apiError = error as ApiError;
				if (apiError.status === 401) {
					this.profile = null;
					this.status = "guest";
					return;
				}

				this.profile = null;
				this.status = "guest";
			} finally {
				this.isLoading = false;
			}
		},

		async login(email: string, password: string): Promise<void> {
			this.isLoading = true;
			const { login, me } = useAuth();

			try {
				await login({ email, password });
				const profile = await me();
				this.profile = profile;
				this.status = "authenticated";
			} finally {
				this.isLoading = false;
			}
		},

		async signup(name: string, email: string, password: string): Promise<void> {
			this.isLoading = true;
			const { signup } = useAuth();

			try {
				await signup({ name, email, password });
			} finally {
				this.isLoading = false;
			}
		},

		async logout(): Promise<void> {
			this.isLoading = true;
			const { logout } = useAuth();

			try {
				await logout();
			} finally {
				this.profile = null;
				this.status = "guest";
				this.isLoading = false;
			}
		},

		async updateProfile(payload: {
			name?: string;
			email?: string;
			password?: string;
			currentPassword?: string;
		}): Promise<void> {
			this.isLoading = true;
			const { updateMe } = useAuth();

			try {
				const profile = await updateMe(payload);
				this.profile = profile;
				this.status = "authenticated";
			} finally {
				this.isLoading = false;
			}
		},
	},
});
