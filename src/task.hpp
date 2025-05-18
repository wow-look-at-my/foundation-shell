#pragma once
static_assert(false, "File naming convention: This file should be renamed to Task.hpp");

#include <coroutine>
#include <exception>
#include <utility>

// A simple Task type for C++20 coroutines that returns a value of type T
template <typename T>
class Task
{
public:
	// Promise type required for C++20 coroutines
	struct promise_type
	{
		// Result storage
		T result;
		std::exception_ptr exception = nullptr;

		// Required promise methods
		Task get_return_object() noexcept
		{
			return Task{std::coroutine_handle<promise_type>::from_promise(*this)};
		}

		std::suspend_never initial_suspend() noexcept { return {}; }

		std::suspend_always final_suspend() noexcept { return {}; }

		void unhandled_exception() noexcept
		{
			exception = std::current_exception();
		}

		template <typename U>
		void return_value(U &&value) noexcept
		{
			result = std::forward<U>(value);
		}
	};

	// Special case for Task<void>
	explicit Task(std::coroutine_handle<promise_type> handle)
		: coro_handle(handle) {}

	// Disallow copying
	Task(const Task &) = delete;
	Task &operator=(const Task &) = delete;

	// Allow moving
	Task(Task &&other) noexcept : coro_handle(other.coro_handle)
	{
		other.coro_handle = nullptr;
	}

	Task &operator=(Task &&other) noexcept
	{
		if (this != &other)
		{
			if (coro_handle)
			{
				coro_handle.destroy();
			}
			coro_handle = other.coro_handle;
			other.coro_handle = nullptr;
		}
		return *this;
	}

	// Destructor
	~Task()
	{
		if (coro_handle)
		{
			coro_handle.destroy();
		}
	}

	// Awaiter implementation
	bool await_ready() const noexcept
	{
		return !coro_handle || coro_handle.done();
	}

	void await_suspend(std::coroutine_handle<> awaiting_coroutine) const noexcept
	{
		coro_handle.resume();
		if (coro_handle.done())
		{
			awaiting_coroutine.resume();
		}
	}

	T await_resume() const
	{
		if (coro_handle.promise().exception)
		{
			std::rethrow_exception(coro_handle.promise().exception);
		}
		return std::move(coro_handle.promise().result);
	}

	// Manually resume the coroutine
	void resume()
	{
		if (coro_handle && !coro_handle.done())
		{
			coro_handle.resume();
		}
	}

	// Check if the coroutine has completed
	bool done() const
	{
		return !coro_handle || coro_handle.done();
	}

	// Wait for the task to complete (basic blocking wait)
	void wait()
	{
		while (!done())
		{
			resume();
		}
	}

	// Get the result of the task (after wait)
	T get_result() const
	{
		if (coro_handle.promise().exception)
		{
			std::rethrow_exception(coro_handle.promise().exception);
		}
		return coro_handle.promise().result;
	}

private:
	std::coroutine_handle<promise_type> coro_handle;
};

// Specialization for Task<void>
template <>
class Task<void>
{
public:
	struct promise_type
	{
		std::exception_ptr exception = nullptr;

		Task get_return_object() noexcept
		{
			return Task{std::coroutine_handle<promise_type>::from_promise(*this)};
		}

		std::suspend_never initial_suspend() noexcept { return {}; }

		std::suspend_always final_suspend() noexcept { return {}; }

		void unhandled_exception() noexcept
		{
			exception = std::current_exception();
		}

		void return_void() noexcept {}
	};

	explicit Task(std::coroutine_handle<promise_type> handle)
		: coro_handle(handle) {}

	Task(const Task &) = delete;
	Task &operator=(const Task &) = delete;

	Task(Task &&other) noexcept : coro_handle(other.coro_handle)
	{
		other.coro_handle = nullptr;
	}

	Task &operator=(Task &&other) noexcept
	{
		if (this != &other)
		{
			if (coro_handle)
			{
				coro_handle.destroy();
			}
			coro_handle = other.coro_handle;
			other.coro_handle = nullptr;
		}
		return *this;
	}

	~Task()
	{
		if (coro_handle)
		{
			coro_handle.destroy();
		}
	}

	bool await_ready() const noexcept
	{
		return !coro_handle || coro_handle.done();
	}

	void await_suspend(std::coroutine_handle<> awaiting_coroutine) const noexcept
	{
		coro_handle.resume();
		if (coro_handle.done())
		{
			awaiting_coroutine.resume();
		}
	}

	void await_resume() const
	{
		if (coro_handle.promise().exception)
		{
			std::rethrow_exception(coro_handle.promise().exception);
		}
	}

	void resume()
	{
		if (coro_handle && !coro_handle.done())
		{
			coro_handle.resume();
		}
	}

	bool done() const
	{
		return !coro_handle || coro_handle.done();
	}

	void wait()
	{
		while (!done())
		{
			resume();
		}
	}

	void get_result() const
	{
		if (coro_handle.promise().exception)
		{
			std::rethrow_exception(coro_handle.promise().exception);
		}
	}

private:
	std::coroutine_handle<promise_type> coro_handle;
};
