import { ref } from "vue";

export interface PaginatedResponse<T> {
	items: T[];
	hasNextPage: boolean;
	hasPreviousPage: boolean;
	startCursor: string;
	endCursor: string;
}

export interface UseCursorPaginationOptions<T> {
	fetchFn: (after: string) => Promise<PaginatedResponse<T>>;
	initialPageSize?: 10 | 20 | 50;
}

export function useCursorPagination<T>(options: UseCursorPaginationOptions<T>) {
	const { fetchFn, initialPageSize = 10 } = options;

	const items = ref<T[]>([]);
	const hasNextPage = ref(false);
	const hasPreviousPage = ref(false);
	const startCursor = ref("");
	const endCursor = ref("");

	const pageSize = ref<10 | 20 | 50>(initialPageSize);
	const currentAfter = ref("");
	const cursorHistory = ref<string[]>([]);
	const isLoading = ref(false);
	const errorMessage = ref("");

	async function fetch(after = currentAfter.value): Promise<void> {
		errorMessage.value = "";
		isLoading.value = true;

		try {
			let requestedAfter = after;
			let hasFallenBack = false;

			while (true) {
				const response = await fetchFn(requestedAfter);

				if (!response || response.items == null) {
					items.value = [];
					hasNextPage.value = false;
					hasPreviousPage.value = false;
					startCursor.value = "";
					endCursor.value = "";
					currentAfter.value = requestedAfter;
					break;
				}

				if (
					response.items.length === 0 &&
					requestedAfter !== "" &&
					cursorHistory.value.length > 0 &&
					!hasFallenBack
				) {
					requestedAfter = cursorHistory.value.pop() ?? "";
					hasFallenBack = true;
					continue;
				}

				items.value = response.items;
				hasNextPage.value = response.hasNextPage;
				hasPreviousPage.value = response.hasPreviousPage;
				startCursor.value = response.startCursor;
				endCursor.value = response.endCursor;
				currentAfter.value = requestedAfter;
				break;
			}
		} catch (error) {
			const apiError = error as { message?: string };
			errorMessage.value = apiError.message || "Failed to load data.";
		} finally {
			isLoading.value = false;
		}
	}

	async function goNext(): Promise<void> {
		if (!hasNextPage.value || endCursor.value.trim() === "") {
			return;
		}

		cursorHistory.value.push(currentAfter.value);
		await fetch(endCursor.value);
	}

	async function goPrevious(): Promise<void> {
		if (cursorHistory.value.length === 0) {
			return;
		}

		const previousAfter = cursorHistory.value.pop() ?? "";
		await fetch(previousAfter);
	}

	async function changePageSize(value: 10 | 20 | 50): Promise<void> {
		pageSize.value = value;
		cursorHistory.value = [];
		await fetch("");
	}

	function resetPagination(): void {
		cursorHistory.value = [];
		currentAfter.value = "";
	}

	return {
		items,
		hasNextPage,
		hasPreviousPage,
		startCursor,
		endCursor,
		pageSize,
		currentAfter,
		cursorHistory,
		isLoading,
		errorMessage,
		fetch,
		goNext,
		goPrevious,
		changePageSize,
		resetPagination,
	};
}