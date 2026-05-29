import { defineStore } from "pinia";

interface OrdersSyncState {
	refreshNonce: number;
}

export const useOrdersSyncStore = defineStore("orders-sync", {
	state: (): OrdersSyncState => ({
		refreshNonce: 0,
	}),
	actions: {
		bumpRefresh(): void {
			this.refreshNonce += 1;
		},
	},
});