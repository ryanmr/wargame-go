package main

import (
	"fmt"
	"time"
	"math"
	"math/rand"
	"strings"
)

// Benchmark accepts a number of threads,
// and will eventually benchmark.
func Benchmark(threads int) {

	progress_channels := make([](chan int), threads)

	create_threads(threads, &progress_channels)

	const ms int64 = 1000000
	const ns int64 = 1000000000

	// 10 seconds
	const prime_time int64 = 	10000000000
	// 60 seconds
	const sample_time int64 = 60000000000

	// 1/15 of a second
	const display_frequency int64 = ns/15
	// 1/200 of a second
	const sample_frequency int64 = 	ns/200

	// when to end the the benchmark
	const end_time int64 = prime_time + sample_time

	const sample_size int64 = sample_time / sample_frequency

	var samples []float64 = make([]float64, 0, sample_size)

	var start_time int64 = time.Now().UnixNano()
	var current_time int64 = time.Now().UnixNano()

	var last_display_time int64 = current_time
	var last_sample_time int64 = current_time

	var elapsed_time int64 = 0

	var phase int = 1
	var total_games int64 = 1

	var speed float64 = 0.0
	var speed_v float64 = 0.0
	var rate float64 = 0.0

	var maximum_speed float64
	var minimum_speed float64

	monitor: for true {
		total_games += int64(collect_progress(&progress_channels))

		current_time = time.Now().UnixNano()
		elapsed_time = current_time - start_time

		rate = float64(elapsed_time) / float64(total_games)

		speed = 1.0 / rate
		speed_v = speed * float64(ms)

		if phase == 1 && elapsed_time >= prime_time {
			phase = 2

			maximum_speed = speed
			minimum_speed = speed

		} else if phase == 2 {

			if maximum_speed < speed {
				maximum_speed = speed
			}

			if minimum_speed > speed {
				minimum_speed = speed
			}

			if elapsed_time >= end_time {
				phase = 3
			}

		} else if phase == 4 {
			break monitor
		}

		if phase == 2 && (current_time - last_sample_time) > sample_frequency {
			last_sample_time = current_time
			samples = append(samples, speed)
		}

		if (current_time - last_display_time) > display_frequency {
			last_display_time = current_time

			if phase == 1 {
				fmt.Printf("\r%d. priming | et = %ds; g = %d; s = %.5f g/ms; \t",
				phase, elapsed_time / ns, total_games, speed_v)
			} else if phase == 2 {
				fmt.Printf("\r%d. sampling | et = %ds; g = %d; s = %.5f g/ms; t = %d; \t",
				phase, elapsed_time / ns, total_games, speed_v, len(samples))
			} else if phase == 3 {
				phase = 4
				fmt.Printf("\r%d. done | et = %ds; g = %d; s = %.5f g/ms; t = %d; \t",
				phase, elapsed_time / ns, total_games, speed_v, len(samples))
			}
		}
	}

	// final statistics
	var mean float64 = get_mean(samples)
	var stdev float64 = get_standard_deviation(samples, mean)
	var cov float64 = get_coefficient_of_variation(mean, stdev)

	// the delta of the max-min speeds
	var min_max_delta float64 = maximum_speed - minimum_speed

	// one_sigma is 1 standard deviation away from the mean
	var one_sigma_lower float64 = (mean-stdev)*float64(ms)
	var one_sigma_upper float64 = (mean+stdev)*float64(ms)
	var one_sigma_delta float64 = one_sigma_upper - one_sigma_lower

	const t_score = 3.291 // 99.9% t-score
	const one_percent = .01 // 1%

	// 99.9% confidence interval; how likely it is that the true mean lies within
	var ci_lower float64 = (mean - (t_score * (stdev / math.Sqrt(float64(len(samples)))))) * float64(ms)
	var ci_upper float64 = (mean + (t_score * (stdev / math.Sqrt(float64(len(samples)))))) * float64(ms)
	var ci_delta float64 = ci_upper - ci_lower

	// controversial section! points are given
	// based on passing basic statistical testing criteria
	var points []string = make([]string, 0, 3)

	// pass: COV < 1%; stdev / mean
	if cov < one_percent {
		points = append(points, "1%cov")
	}

	// the final speed is within 1 stdev
	if one_sigma_lower < speed_v && speed_v < one_sigma_upper {
		points = append(points, "1σ")
	}

	// the final speed is near the true mean
	if ci_lower < speed_v && speed_v < ci_upper {
		points = append(points, "99.9%CI")
	}


	fmt.Printf("\n---\n")

	fmt.Printf("Samples: %d collected\n", len(samples))
	fmt.Printf("Mean: %.5f\n", mean * float64(ms))
	fmt.Printf("Standard Deviation: %.5f\n", stdev * float64(ms))
	fmt.Printf("Coefficient of Variation: %.5f\n", cov)

	fmt.Printf("Min-Max:\t < %.5f - %.5f > Δ %.5f\n", minimum_speed*float64(ms), maximum_speed*float64(ms), min_max_delta*float64(ms))

	fmt.Printf("1-σ:\t\t < %.5f - %.5f > Δ %.5f\n", one_sigma_lower, one_sigma_upper, one_sigma_delta)

	fmt.Printf("99.9%% CI:\t < %.5f - %.5f > Δ %.5f", ci_lower, ci_upper, ci_delta)


	fmt.Printf("\n---\n")

	fmt.Printf("Threads: %d\n", threads)
	fmt.Printf("Speed: %.5f\n", speed_v)
	fmt.Printf("Total Games: %d\n", total_games)
	fmt.Printf("Elapsed Time: %.0f seconds\n", float64(elapsed_time / ns))

	fmt.Printf("Rank Passes: %s\n", rank_reason(points))
	fmt.Printf("\nScore: %d %s\n", math_round(speed_v), rank_letter(points))

}

// Rank letter accepts a list of passed tests that
// define certain statistical qualities.
// For each successful pass, a better letter rank is returned.
func rank_letter(passes []string) string {
	v := len(passes)
	letter := ""
	switch v {
		case 3: letter = "A"
		case 2: letter = "B"
		case 1: letter = "C"
		case 0: letter = "D"
		default: letter = "X"
	}
	return letter
}

// Rank reason concatenates a string, or reports none.
// The rank reasons are set in a string slice.
func rank_reason(passes []string) string {
	reason := ""
	if len(passes) == 0 {
		reason = "none"
	} else {
		reason = strings.Join(passes[:], ", ")
	}
	return reason
}

// Math round attempts to round a float to an integer.
func math_round(f float64) int64 {
	return int64(math.Floor(f + .5))
}

// Create threads accepts a number of threads and a channel pointer,
// which will be populated with channels for each respective thread
// spun up. Each thread runs a game loop, and upon completion of each game,
// the thread sends through the progress channel.
func create_threads(threads int, channels *[](chan int)) {
	for i := 0; i < threads; i++ {

		progress := make(chan int, threads * 1024)
		(*channels)[i] = progress

		go func() {
			source := rand.NewSource(time.Now().UnixNano())
			generator := rand.New(source)
			for true {
				Game(generator)
				progress <- 1
			}
		}()

	}
}

// Collect progress accesses each channel and listens, in a non-blocking fashion,
// for any data passed back to it that indicates progress has been made.
func collect_progress(channels *[](chan int)) int {
	r := 0
	for _,v := range *channels {
		select {
			case p := <-v:
				r += p
			default:
				// no data available
		}
	}
	return r
}

// Get mean calculates the mean based on the given samples.
func get_mean(samples []float64) float64 {
	var total float64 = 0
	for _, v := range samples {
		total += v
	}
	var mean float64 = total / float64(len(samples))
	return mean
}

// Get standard deviation calculates the stdev based on the given samples and the mean.
// TODO: implement `online_variance` algorithm
func get_standard_deviation(samples []float64, mean float64) float64 {
	var total float64 = 0
	for _, v := range samples {
		total += math.Pow(v - mean, 2)
	}
	var stdev float64 = math.Sqrt(total / float64(len(samples)))
	return stdev
}

// Get coefficient of variation calculations the standard deviation to mean ratio.
func get_coefficient_of_variation(mean float64, stdev float64) float64 {
	return stdev / mean
}
