import os
import time
import requests
import pandas as pd
from prophet import Prophet
from prometheus_remote_writer import RemoteWriter

writer = RemoteWriter(
    url='http://localhost:8428/api/v1/write', # write to victoria
)
PROM_URL = 'http://localhost:9090'
QUERY = os.environ.get(
    "QUERY", 'http_request_latency'
)
LOOKBACK = os.environ.get('LOOKBACK', '15m')
PREDICT_PERIOD = 5 * 60 # seconds
METRIC_NAME = 'http_request_latency_forecast'

def query_prometheus(query, lookback):
    """
    Queries Prometheus for time series data.

    Args:
        query (str): Prometheus query string.
        lookback (str):  Lookback duration string (e.g., '7d').

    Returns:
        pandas.DataFrame: Time series data with 'ds' (datetime) and 'y' (value) columns.
    """
    end = time.time()
    start = end - pd.Timedelta(lookback).total_seconds()
    query_url = f"{PROM_URL}/api/v1/query_range"
    params = {"query": query, "start": start, "end": end, "step": "1"}  # Step is important, adjust based on data frequency
    try:
        response = requests.get(query_url, params=params)
        response.raise_for_status()
        data = response.json()

        if data["status"] == "success":
            results = data["data"]["result"]
            if not results:
                print("Warning: No data returned from Prometheus.")
                return pd.DataFrame()

            # Assuming we only want the first result in the result set.  Adjust if needed.
            values = results[0]["values"]  # Each element is [timestamp, value]
            df = pd.DataFrame(values, columns=["ds", "y"])
            df["ds"] = pd.to_datetime(df["ds"], unit="s")  # Convert timestamp to datetime
            df["y"] = pd.to_numeric(df["y"])
            return df
        else:
            print(f"Error: Prometheus query failed: {data['error']}")
            return pd.DataFrame()
    except requests.exceptions.RequestException as e:
        print(f"Error:  Prometheus query failed with exception: {e}")
        return pd.DataFrame()


def train_prophet_model(df, predict_period):
    """
    Trains a Prophet model on the given data and makes predictions.

    Args:
        df (pandas.DataFrame): Time series data with 'ds' (datetime) and 'y' (value) columns.
        predict_period (int): Prediction horizon in hours.

    Returns:
        pandas.DataFrame: Forecasted values with 'ds' (datetime) and 'yhat' (forecast) columns.  Returns empty DataFrame on error.
    """
    try:
        model = Prophet(interval_width=0.95)
        model.fit(df)
        future = model.make_future_dataframe(periods=predict_period, freq="s")  # Predict for the next X minutes
        forecast = model.predict(future)

        forecast = forecast.tail(predict_period)
        forecast = forecast[-1:]
        print(forecast)

        return forecast[["ds", "yhat"]]  # Return only the needed columns
    except Exception as e:
        print(f"Error: Prophet model training failed: {e}")
        return pd.DataFrame()  # Return an empty DataFrame on error


def chunker(seq, size):
    """Helper function to yield chunks of a list."""
    return (seq[pos:pos + size] for pos in range(0, len(seq), size))

def write_forecast_to_prometheus(forecast, metric_name):
    """
    Writes the forecasted values to Prometheus using the remote write endpoint or pushgateway.

    Args:
        forecast (pandas.DataFrame): Forecasted values with 'ds' (datetime) and 'yhat' (forecast) columns.
        metric_name (str): The name of the metric to store in Prometheus.

    Raises:
        ValueError: If writer is None and pushgateway is not properly set up.
    """
    CHUNK_SIZE = 10  # Added constant for chunking

    try:
        data = []
        for index, row in forecast.iterrows():
            timestamp = int(row["ds"].timestamp() * 1000)  # Convert to milliseconds
            ### Dirty hack
            ### Remote write can not accept values too far in the future
            ### But only for prometheus, victora works differently
            # timestamp = timestamp - PREDICT_PERIOD * 5 * 1000
            # print(timestamp)
            value = row["yhat"]

            # Construct the Prometheus data point in the correct format
            data_point = {
                "metric": {
                    "__name__": metric_name,  # Using the passed-in metric_name
                    "job": "forecast",      # Add any other labels here
                },
                "values": [value],
                "timestamps": [timestamp],
            }
            data.append(data_point)

        # Send the data in chunks
        for chunk in chunker(data, CHUNK_SIZE):
            try:
                writer.send(chunk)
                print(f"Sent chunk of size: {len(chunk)} to remote write endpoint")
            except Exception as e:
                print(f"Error sending chunk: {e}")
    except Exception as e:
        print(f"Error: Error preparing data for Prometheus: {e}")


def main():
    """
    Main function to orchestrate the data retrieval, model training, and forecast writing.
    """
    print("Starting Prophet forecasting...")

    while True:
        try:
            print("Querying Prometheus...")
            data = query_prometheus(QUERY, LOOKBACK)

            if not data.empty:
                print("Training Prophet model...")
                forecast = train_prophet_model(data, PREDICT_PERIOD)

                if not forecast.empty:
                    print("Writing forecast to Prometheus...")
                    write_forecast_to_prometheus(forecast, METRIC_NAME)
                else:
                    print("Skipping writing forecast due to empty forecast data.")
            else:
                print("Skipping training and writing forecast due to empty data.")

            print(f"Sleeping ...")
            time.sleep(1)

        except Exception as e:
            print(f"An error occurred: {e}")
            time.sleep(1)


if __name__ == "__main__":
    main()