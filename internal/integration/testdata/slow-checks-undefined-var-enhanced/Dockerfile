# Test: undefined var false positive elimination via async check.
# PYTHON_VERSION is set in the real python image env, so $PYTHON_VERSION should
# NOT be flagged as undefined when slow checks resolve the actual image env.
FROM python:3.12
RUN echo "Python version: $PYTHON_VERSION"
